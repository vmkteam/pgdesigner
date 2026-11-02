package sql

import (
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	pgquery "github.com/wasilibs/go-pgquery"
)

// ExprKey returns a key for comparing SQL expressions that differ only in how they are
// written. An expression from the database (pg_get_expr) has extra parentheses, casts on
// constants and implicit casts (varchar to text, integer to numeric), negative numbers as
// '-1'::integer, IN lists as = ANY (ARRAY[...]), LIKE as ~~ and BETWEEN as two comparisons;
// CURRENT_TIMESTAMP is the same as now().
// The key is for comparison only. Explicit casts to other types and string literals are kept,
// so changing them changes the key. On a parse error the trimmed expression is returned.
func ExprKey(expr string) string {
	expr = strings.TrimSpace(expr)
	n := parseExpr(expr)
	if n == nil {
		return expr
	}
	normalizeKey(n)
	if s := nodeToSQL(n); s != "" {
		return s
	}
	return expr
}

// parseExpr parses a single SQL expression, or returns nil.
func parseExpr(expr string) *pg.Node {
	tree, err := pgquery.Parse("SELECT " + expr)
	if err != nil || len(tree.Stmts) != 1 {
		return nil
	}
	targets := tree.Stmts[0].GetStmt().GetSelectStmt().GetTargetList()
	if len(targets) != 1 {
		return nil
	}
	return targets[0].GetResTarget().GetVal()
}

// implicitCastTypes are the targets of implicit casts that pg_get_expr writes out, e.g.
// (name)::text for a varchar column passed to a text function.
var implicitCastTypes = map[string]bool{
	"text": true, "varchar": true, "bpchar": true,
	"numeric": true, "int8": true, "float4": true, "float8": true,
}

// normalizeKey rewrites an expression tree in place, children first, see ExprKey.
func normalizeKey(n *pg.Node) {
	switch v := n.GetNode().(type) {
	case *pg.Node_AConst:
		// -1 and '-1'::integer must compare equal: keep numbers as strings
		switch {
		case v.AConst.GetIval() != nil:
			n.Node = pg.MakeAConstStrNode(strconv.Itoa(int(v.AConst.GetIval().GetIval())), 0).Node
		case v.AConst.GetFval() != nil:
			n.Node = pg.MakeAConstStrNode(v.AConst.GetFval().GetFval(), 0).Node
		}
	case *pg.Node_SqlvalueFunction:
		if v.SqlvalueFunction.GetOp() == pg.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP {
			n.Node = &pg.Node_FuncCall{FuncCall: &pg.FuncCall{Funcname: []*pg.Node{pg.MakeStrNode("now")}}}
		}
	case *pg.Node_TypeCast:
		arg := v.TypeCast.GetArg()
		normalizeKey(arg)
		if isConstant(arg) || isImplicitCast(v.TypeCast.GetTypeName()) {
			n.Node = arg.GetNode()
		}
	case *pg.Node_AExpr:
		normalizeKey(v.AExpr.GetLexpr())
		normalizeKey(v.AExpr.GetRexpr())
		normalizeAExpr(n, v.AExpr)
	case *pg.Node_BoolExpr:
		normalizeKeys(v.BoolExpr.GetArgs())
		v.BoolExpr.Args = flattenBool(v.BoolExpr.GetBoolop(), v.BoolExpr.GetArgs())
	case *pg.Node_FuncCall:
		normalizeKeys(v.FuncCall.GetArgs())
	case *pg.Node_AArrayExpr:
		normalizeKeys(v.AArrayExpr.GetElements())
	case *pg.Node_List:
		normalizeKeys(v.List.GetItems())
	case *pg.Node_CoalesceExpr:
		normalizeKeys(v.CoalesceExpr.GetArgs())
	case *pg.Node_MinMaxExpr:
		normalizeKeys(v.MinMaxExpr.GetArgs())
	case *pg.Node_RowExpr:
		normalizeKeys(v.RowExpr.GetArgs())
	case *pg.Node_CaseExpr:
		normalizeKey(v.CaseExpr.GetArg())
		normalizeKeys(v.CaseExpr.GetArgs())
		normalizeKey(v.CaseExpr.GetDefresult())
	case *pg.Node_CaseWhen:
		normalizeKey(v.CaseWhen.GetExpr())
		normalizeKey(v.CaseWhen.GetResult())
	case *pg.Node_NullTest:
		normalizeKey(v.NullTest.GetArg())
	case *pg.Node_BooleanTest:
		normalizeKey(v.BooleanTest.GetArg())
	case *pg.Node_AIndirection:
		normalizeKey(v.AIndirection.GetArg())
	case *pg.Node_CollateClause:
		normalizeKey(v.CollateClause.GetArg())
	}
}

func normalizeKeys(nodes []*pg.Node) {
	for _, n := range nodes {
		normalizeKey(n)
	}
}

// normalizeAExpr rewrites an operator expression the way PostgreSQL stores it.
func normalizeAExpr(n *pg.Node, e *pg.A_Expr) {
	op := funcName(e.GetName())
	switch e.GetKind() {
	case pg.A_Expr_Kind_AEXPR_OP_ANY, pg.A_Expr_Kind_AEXPR_OP_ALL:
		// x = ANY (ARRAY[...]) is how x IN (...) is stored, x <> ALL (ARRAY[...]) is x NOT IN (...)
		arr := e.GetRexpr().GetAArrayExpr()
		if arr == nil || op != "=" && e.GetKind() == pg.A_Expr_Kind_AEXPR_OP_ANY || op != "<>" && e.GetKind() == pg.A_Expr_Kind_AEXPR_OP_ALL {
			return
		}
		e.Kind = pg.A_Expr_Kind_AEXPR_IN
		e.Rexpr = pg.MakeListNode(arr.GetElements())
		normalizeAExpr(n, e)
	case pg.A_Expr_Kind_AEXPR_IN:
		// x IN ('a') is stored as x = 'a'
		if items := e.GetRexpr().GetList().GetItems(); len(items) == 1 {
			e.Kind, e.Rexpr = pg.A_Expr_Kind_AEXPR_OP, items[0]
		}
	case pg.A_Expr_Kind_AEXPR_LIKE, pg.A_Expr_Kind_AEXPR_ILIKE:
		e.Kind = pg.A_Expr_Kind_AEXPR_OP // the name is already ~~, !~~, ~~* or !~~*
	case pg.A_Expr_Kind_AEXPR_BETWEEN, pg.A_Expr_Kind_AEXPR_NOT_BETWEEN:
		bounds := e.GetRexpr().GetList().GetItems()
		if len(bounds) != 2 {
			return
		}
		boolop, lo, hi := pg.BoolExprType_AND_EXPR, ">=", "<="
		if e.GetKind() == pg.A_Expr_Kind_AEXPR_NOT_BETWEEN {
			boolop, lo, hi = pg.BoolExprType_OR_EXPR, "<", ">"
		}
		n.Node = pg.MakeBoolExprNode(boolop, []*pg.Node{
			pg.MakeAExprNode(pg.A_Expr_Kind_AEXPR_OP, []*pg.Node{pg.MakeStrNode(lo)}, e.GetLexpr(), bounds[0], 0),
			pg.MakeAExprNode(pg.A_Expr_Kind_AEXPR_OP, []*pg.Node{pg.MakeStrNode(hi)}, e.GetLexpr(), bounds[1], 0),
		}, 0).Node
	}
}

// flattenBool merges nested AND (OR) arguments into one list: a AND (b AND c) is a AND b AND c.
func flattenBool(op pg.BoolExprType, args []*pg.Node) []*pg.Node {
	if op == pg.BoolExprType_NOT_EXPR {
		return args
	}
	var res []*pg.Node
	for _, a := range args {
		if b := a.GetBoolExpr(); b != nil && b.GetBoolop() == op {
			res = append(res, b.GetArgs()...)
		} else {
			res = append(res, a)
		}
	}
	return res
}

// isConstant reports whether n is a constant or an array of constants.
func isConstant(n *pg.Node) bool {
	if n.GetAConst() != nil {
		return true
	}
	arr := n.GetAArrayExpr()
	if arr == nil {
		return false
	}
	for _, e := range arr.GetElements() {
		if !isConstant(e) {
			return false
		}
	}
	return true
}

// isImplicitCast reports whether a cast may be one that pg_get_expr writes for an implicit
// coercion: a cast without type modifiers to one of implicitCastTypes or an array of them.
func isImplicitCast(t *pg.TypeName) bool {
	names := t.GetNames()
	if len(names) == 0 || len(t.GetTypmods()) > 0 {
		return false
	}
	return implicitCastTypes[strings.ToLower(names[len(names)-1].GetString_().GetSval())]
}

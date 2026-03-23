package pgd

// MigratePartitions converts Variant A (separate tables with PartitionOf) to Variant B
// (children nested inside parent as Partition elements). Should be called after SQL parsing
// and after reverse engineering.
func MigratePartitions(p *Project) {
	for i := range p.Schemas {
		migrateSchemaPartitions(&p.Schemas[i])
	}
}

func migrateSchemaPartitions(s *Schema) {
	childMap := map[string][]Table{} // parentName → children
	var regular []Table

	for _, t := range s.Tables {
		if t.PartitionOf != "" {
			childMap[t.PartitionOf] = append(childMap[t.PartitionOf], t)
		} else {
			regular = append(regular, t)
		}
	}

	if len(childMap) == 0 {
		return
	}

	for i, t := range regular {
		children, ok := childMap[t.Name]
		if !ok {
			continue
		}
		for _, child := range children {
			p := Partition{Name: child.Name}
			if child.PartitionBound != nil {
				p.Bound = child.PartitionBound.Value
			}
			if child.PartitionBy != nil {
				p.PartitionBy = child.PartitionBy
			}
			if child.Tablespace != "" {
				p.Tablespace = child.Tablespace
			}
			if child.With != nil {
				p.With = child.With
			}
			regular[i].Partitions = append(regular[i].Partitions, p)
		}
	}

	s.Tables = regular
}

// CollectPartitionChildren returns a set of table names that are partition children
// (Variant B: nested inside parent). Used to skip index/FK generation for children.
func CollectPartitionChildren(p *Project) map[string]bool {
	children := make(map[string]bool)
	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			collectPartChildren(&t, children)
		}
	}
	return children
}

func collectPartChildren(t *Table, children map[string]bool) {
	for i := range t.Partitions {
		children[t.Partitions[i].Name] = true
		collectPartChildrenRec(&t.Partitions[i], children)
	}
}

func collectPartChildrenRec(p *Partition, children map[string]bool) {
	for i := range p.Partitions {
		children[p.Partitions[i].Name] = true
		collectPartChildrenRec(&p.Partitions[i], children)
	}
}

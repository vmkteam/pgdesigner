import { describe, expect, it } from 'vitest'
import { pkColumnName, toCamelCase, toPascalCase, toSnakeCase } from '../naming'

describe('toSnakeCase', () => {
  it.each([
    ['order', 'order'],
    ['Order', 'order'],
    ['orderItem', 'order_item'],
    ['OrderItem', 'order_item'],
    ['order_item', 'order_item'],
    ['order-item', 'order_item'],
    ['HTTPRequest', 'http_request'],
    ['userID', 'user_id'],
    ['', ''],
  ])('toSnakeCase(%j) -> %j', (input, expected) => {
    expect(toSnakeCase(input)).toBe(expected)
  })
})

describe('toCamelCase', () => {
  it.each([
    ['order', 'order'],
    ['Order', 'order'],
    ['order_item', 'orderItem'],
    ['OrderItem', 'orderItem'],
    ['orderItem', 'orderItem'],
    ['HTTP_REQUEST', 'httpRequest'],
    ['', ''],
  ])('toCamelCase(%j) -> %j', (input, expected) => {
    expect(toCamelCase(input)).toBe(expected)
  })
})

describe('toPascalCase', () => {
  it.each([
    ['order', 'Order'],
    ['orderItem', 'OrderItem'],
    ['order_item', 'OrderItem'],
    ['HTTP_REQUEST', 'HttpRequest'],
    ['', ''],
  ])('toPascalCase(%j) -> %j', (input, expected) => {
    expect(toPascalCase(input)).toBe(expected)
  })
})

describe('pkColumnName', () => {
  it.each([
    ['order', 'snake_case', 'order_id'],
    ['order', 'camelCase', 'orderId'],
    ['order', 'PascalCase', 'OrderId'],
    ['OrderItem', 'snake_case', 'order_item_id'],
    ['OrderItem', 'camelCase', 'orderItemId'],
    ['OrderItem', 'PascalCase', 'OrderItemId'],
    ['order_item', 'snake_case', 'order_item_id'],
    ['order_item', 'camelCase', 'orderItemId'],
    ['order_item', 'PascalCase', 'OrderItemId'],
    ['order', '', 'order_id'],
    ['OrderItem', 'unknown', 'order_item_id'],
  ])('pkColumnName(%j, %j) -> %j', (singular, naming, expected) => {
    expect(pkColumnName(singular, naming)).toBe(expected)
  })
})

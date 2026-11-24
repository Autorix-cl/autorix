import jsep, { type Expression } from "jsep";

interface LogicalExpressionNode extends Expression {
  type: "LogicalExpression";
  operator: string;
  left: Expression;
  right: Expression;
}

const UNSAFE_PROPERTY_NAMES = new Set(["constructor", "__proto__", "prototype"]);

export function evalCel(expression: string, context: Record<string, unknown>): boolean {
  try {
    const ast = jsep(expression);
    const result = evaluateNode(ast, context);
    return Boolean(result);
  } catch (error) {
    if (error instanceof Error && error.message.includes("Unsafe")) {
      throw error;
    }
    return false;
  }
}

function evaluateNode(node: Expression, context: Record<string, unknown>): unknown {
  switch (node.type) {
    case "Literal":
      return (node as jsep.Literal).value;

    case "Identifier": {
      const idName = (node as jsep.Identifier).name;
      if (UNSAFE_PROPERTY_NAMES.has(idName)) {
        throw new Error("Unsafe identifier");
      }
      return Object.prototype.hasOwnProperty.call(context, idName) ? context[idName] : undefined;
    }

    case "MemberExpression": {
      const memNode = node as jsep.MemberExpression;
      const target = evaluateNode(memNode.object, context);

      if (!target || typeof target !== "object") return undefined;

      const property = memNode.computed
        ? String(evaluateNode(memNode.property, context))
        : (memNode.property as jsep.Identifier).name;

      if (UNSAFE_PROPERTY_NAMES.has(property)) {
        throw new Error("Unsafe property access");
      }

      const record = target as Record<string, unknown>;
      return Object.prototype.hasOwnProperty.call(record, property) ? record[property] : undefined;
    }

    case "BinaryExpression": {
      const binNode = node as jsep.BinaryExpression;
      if (binNode.operator === "&&") {
        return evaluateNode(binNode.left, context) && evaluateNode(binNode.right, context);
      }
      if (binNode.operator === "||") {
        return evaluateNode(binNode.left, context) || evaluateNode(binNode.right, context);
      }

      const left = evaluateNode(binNode.left, context);
      const right = evaluateNode(binNode.right, context);

      switch (binNode.operator) {
        case "==":
        case "===":
          return left === right;
        case "!=":
        case "!==":
          return left !== right;
        case ">":
          return Number(left) > Number(right);
        case ">=":
          return Number(left) >= Number(right);
        case "<":
          return Number(left) < Number(right);
        case "<=":
          return Number(left) <= Number(right);
        default:
          throw new Error(`Unsupported operator: ${binNode.operator}`);
      }
    }

    case "LogicalExpression": {
      const logNode = node as LogicalExpressionNode;
      const left = evaluateNode(logNode.left, context);

      if (logNode.operator === "&&") {
        return left && evaluateNode(logNode.right, context);
      }
      if (logNode.operator === "||") {
        return left || evaluateNode(logNode.right, context);
      }
      throw new Error(`Unsupported logical operator: ${logNode.operator}`);
    }

    case "UnaryExpression": {
      const unNode = node as jsep.UnaryExpression;
      const argument = evaluateNode(unNode.argument, context);
      if (unNode.operator === "!") return !argument;
      if (unNode.operator === "-") return -Number(argument);
      if (unNode.operator === "+") return +Number(argument);
      throw new Error(`Unsupported unary operator: ${unNode.operator}`);
    }

    case "CallExpression":
      throw new Error("Unsafe function call execution");

    default:
      throw new Error(`Unsupported syntax node: ${node.type}`);
  }
}

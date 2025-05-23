package evaluator

import (
	"fmt"
	"mfiorek/waiig/ast"
	"mfiorek/waiig/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {

	// INFO: Statements:
	case *ast.Program:
		return evalProgram(node.Statements, env)
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)
	case *ast.BlockStatement:
		return evalStatements(node.Statements, env)
	case *ast.ReturnStatement:
		returnValueEvaluated := Eval(node.ReturnValue, env)
		if isError(returnValueEvaluated) {
			return returnValueEvaluated
		}
		return &object.ReturnValue{Value: returnValueEvaluated}
	case *ast.LetStatement:
		evaluated := Eval(node.Value, env)
		if isError(evaluated) {
			return evaluated
		}
		env.Set(node.Name.Value, evaluated)

	// INFO: Expressions:
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.PrefixExpression:
		rightEvaluated := Eval(node.Right, env)
		if isError(rightEvaluated) {
			return rightEvaluated
		}
		return evalPrefixExpression(node.Operator, rightEvaluated)
	case *ast.InfixExpression:
		leftEvaluated := Eval(node.Left, env)
		if isError(leftEvaluated) {
			return leftEvaluated
		}
		rightEvaluated := Eval(node.Right, env)
		if isError(rightEvaluated) {
			return rightEvaluated
		}
		return evalInfixExpression(node.Operator, leftEvaluated, rightEvaluated)
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Body: body, Env: env}
	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args)
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)
	case *ast.HashLiteral:
		return evalHashLiteral(node, env)
	}

	return nil
}

// INFO: ==================================== STATEMENTS ====================================

func evalProgram(stmts []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range stmts {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	return result
}

func evalStatements(stmts []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range stmts {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

// INFO: ==================================== EXPRESSIONS ====================================

// INFO: PrefixExpressions:

// TODO: I may want to change this to take token.TokenType as first parameter
// and cases match i.e. token.BANG - would seem cleaner to me
func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

// NOTE: my version (more verbose) of evalBangOperatorExpression (it also assumes that 0 is falsy, while all other integers are truthy):

// func evalBangOperatorExpression(rightEvaluated object.Object) object.Object {
// 	switch evaluatedType := rightEvaluated.(type) {
// 	case *object.Boolean:
// 		if evaluatedType.Value == true {
// 			return FALSE
// 		} else {
// 			return TRUE
// 		}
// 	case *object.Integer:
// 		if evaluatedType.Value == 0 {
// 			return TRUE
// 		} else {
// 			return FALSE
// 		}
// 	case *object.Null:
// 		return TRUE
// 	default:
// 		return FALSE
// 	}
// }

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

// INFO: InfixExpressions:

func evalInfixExpression(operator string, left, right object.Object) object.Object {

	// NOTE: my version would assert if I have correct Object types like this, and have evalIntegerInfixExpression receive object.Integers

	// leftInt, leftOk := left.(*object.Integer)
	// rightInt, rightOk := right.(*object.Integer)
	// if !leftOk || !rightOk {
	// 	switch {
	// 	case operator == "==":
	// 		return nativeBoolToBooleanObject(left == right) // <-- comparing equality between pointers (it works because we always use the same two pointers for booleans)
	// 	case operator == "!=":
	// 		return nativeBoolToBooleanObject(left != right) // <-- comparing equality between pointers (it works because we always use the same two pointers for booleans)
	// 	default:
	// 		return NULL
	// 	}
	// }
	// return evalIntegerInfixExpression(operator, *leftInt, *rightInt)

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right) // <-- comparing equality between pointers (it works because we always use the same two pointers for booleans)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right) // <-- comparing equality between pointers (it works because we always use the same two pointers for booleans)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(operator string, left, right object.Object) object.Object {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftValue + rightValue}
	case "-":
		return &object.Integer{Value: leftValue - rightValue}
	case "*":
		return &object.Integer{Value: leftValue * rightValue}
	case "/":
		return &object.Integer{Value: leftValue / rightValue}
	case "<":
		return nativeBoolToBooleanObject(leftValue < rightValue)
	case ">":
		return nativeBoolToBooleanObject(leftValue > rightValue)
	case "==":
		return nativeBoolToBooleanObject(leftValue == rightValue)
	case "!=":
		return nativeBoolToBooleanObject(leftValue != rightValue)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalStringInfixExpression(operator string, left, right object.Object) object.Object {
	leftValue := left.(*object.String).Value
	rightValue := right.(*object.String).Value

	switch operator {
	case "+":
		return &object.String{Value: leftValue + rightValue}
	case "==":
		return nativeBoolToBooleanObject(leftValue == rightValue)
	case "!=":
		return nativeBoolToBooleanObject(leftValue != rightValue)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

// INFO: IfExpressions:

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

// INFO: Identifiers:

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if value, ok := env.Get(node.Value); ok {
		return value
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return newError("identifier not found: %s", node.Value)
}

// INFO: CallExpressions (not explicitly, but all this is needed for CallExpressions):
// TODO: I think I'd like to create a function evalCallExpression(node *ast.Identifier, env *object.Environment)
// it would call evalExpression, extendFunctionEnv, unwrapReturnValue
// and contain the applyFunction + the case *ast.CallExpression logic

// NOTE: for evaluating function parameters
func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, exp := range exps {
		eval := Eval(exp, env)
		if isError(eval) {
			return []object.Object{eval}
		}
		result = append(result, eval)
	}

	return result
}

// NOTE: evaluating the CallExpression by applying the function
func applyFunction(fn object.Object, args []object.Object) object.Object {

	switch fn := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)
	case *object.Builtin:
		return fn.Fn(args...)
	default:
		return newError("not a function: %s", fn.Type())
	}
}

// WARN: Helper method used only in applyFunction - extending the env of a function by it's parameters
func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
}

// WARN: Helper method used only in applyFunction - needed because in monkey both the last statement and the return statement can be returned
func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	// for implicit returns...
	return obj
}

// INFO: IndexExpression

func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

// WARN: Helper method used only in evalIndexExpression
func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return NULL
	}

	return arrayObject.Elements[idx]
}

// WARN: Helper method used only in evalIndexExpression
func evalHashIndexExpression(hash, key object.Object) object.Object {
	hashObject := hash.(*object.Hash)

	hashableKey, ok := key.(object.Hashable)
	if !ok {
		return newError("unusable as hash key: %s", key.Type())
	}

	pair, ok := hashObject.Pairs[hashableKey.HashKey()]
	if !ok {
		return NULL
	}

	return pair.Value
}

// INFO: HashLiteral

func evalHashLiteral(node *ast.HashLiteral, env *object.Environment) object.Object {
	hash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}

	for key, value := range node.Pairs {
		keyObject := Eval(key, env)
		if isError(keyObject) {
			return keyObject
		}

		hashableKey, ok := keyObject.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", keyObject.Type())
		}
		hashedKey := hashableKey.HashKey()

		valueObject := Eval(value, env)
		if isError(valueObject) {
			return valueObject
		}

		hash.Pairs[hashedKey] = object.HashPair{Key: keyObject, Value: valueObject}
	}

	return hash
}

// INFO: ==================================== Helper methods ====================================

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

func newError(format string, a ...any) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

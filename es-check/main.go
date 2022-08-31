package main

import (
	"fmt"
	"log"

	"github.com/hsiaosiyuan0/mole/ecma/parser"
	"github.com/hsiaosiyuan0/mole/ecma/walk"
	"github.com/hsiaosiyuan0/mole/span"
)

func main() {
	// imitate the source code you want to parse
	code := `async function* f() {
		let a = 1
	}`

	// create a Source instance to handle to the source code
	s := span.NewSource("example.js", code)

	// create a parser, here we use the default options
	opts := parser.NewParserOpts()
	p := parser.NewParser(s, opts)

	// inform the parser do its parsing process
	ast, err := p.Prog()
	if err != nil {
		log.Fatal(err)
	}

	errs := make([]*NotPermittedSyntaxErr, 0)
	ctx := walk.NewWalkCtx(ast, p.Symtab())

	// subscribe the walking events
	walk.AddListener(&ctx.Listeners, walk.N_STMT_VAR_DEC_BEFORE, &walk.Listener{
		Id: "N_STMT_VAR_DEC_BEFORE", // a key string to identify this listener for you can unsubscribe it later
		Handle: func(node parser.Node, key string, ctx *walk.VisitorCtx) {
			n := node.(*parser.VarDecStmt)
			kind := n.Kind()
			if kind != "var" {
				errs = append(errs, &NotPermittedSyntaxErr{p.Source(), fmt.Sprintf("`%s` is not permitted in es5", kind), node})
			}
		},
	})

	// either `funExpr` or `funStmt` will emit the same event `walk.N_EXPR_FN_BEFORE` and `walk.N_EXPR_FN_AFTER`
	// to distinguish `funExpr` or `funStmt` by using below pattern in the listener:
	// - `n.Type() == parser.N_EXPR_FN` for `funExpr`
	// - `n.Type() == parser.N_STMT_FN` for `funStmt`
	walk.AddListener(&ctx.Listeners, walk.N_EXPR_FN_BEFORE, &walk.Listener{
		Id: "N_EXPR_FN_BEFORE", // a key string to identify this listener for you can unsubscribe it later
		Handle: func(node parser.Node, key string, ctx *walk.VisitorCtx) {
			n := node.(*parser.FnDec)
			if n.Async() {
				if n.Generator() {
					errs = append(errs, &NotPermittedSyntaxErr{p.Source(), "async generator is not permitted in es5", node})
					return
				}
				errs = append(errs, &NotPermittedSyntaxErr{p.Source(), "async function is not permitted in es5", node})
			}
		},
	})

	walk.VisitNode(ast, "", ctx.VisitorCtx())

	for _, err := range errs {
		fmt.Println(err.Error())
	}
}

type NotPermittedSyntaxErr struct {
	s    *span.Source
	msg  string
	node parser.Node
}

func (e *NotPermittedSyntaxErr) Error() string {
	loc := e.s.OfstLineCol(e.node.Range().Lo)
	return fmt.Sprintf("%s at %s#%d:%d", e.msg, e.s.Path, loc.Line, loc.Col)
}

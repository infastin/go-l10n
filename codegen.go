package main

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

type Generator interface {
	IsZero() bool
	Generate(loc *Localization, scope *MessageScope, builderName string, list *[]ast.Stmt)
	SimpleGenerate(loc *Localization, scope *MessageScope, list *[]ast.Stmt)
	GetArgumentNames() (args []string)
}

func generateLocalizations(locs []Localization) (files []*ast.File) {
	files = append(files, generateGeneral(locs))

	for i := 0; i < len(locs); i++ {
		files = append(files, generateMessages(&locs[i]))
	}

	return files
}

func generateGeneral(locs []Localization) (file *ast.File) {
	file = &ast.File{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// Code generated by go-l10n; DO NOT EDIT."},
				{Text: ""},
			},
		},
		Name:  ast.NewIdent(config.PackageName),
		Decls: []ast.Decl{},
	}

	if len(config.Imports) != 0 {
		importDecl := &ast.GenDecl{
			Tok: token.IMPORT,
		}

		for _, imp := range config.Imports {
			importDecl.Specs = append(importDecl.Specs, &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(imp.Import),
				},
			})
		}

		file.Decls = append(file.Decls, importDecl)
	}

	file.Decls = append(file.Decls, &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("Localizer"),
				Type: generateGeneralInterface(locs[0].Scopes),
			},
		},
	})

	generateGeneralTable(locs, &file.Decls)
	generateGeneralSupported(locs, &file.Decls)
	generateGeneralFuncs(locs, &file.Decls)

	return file
}

func generateGeneralInterface(msgs []MessageScope) (ifaceType *ast.InterfaceType) {
	ifaceType = &ast.InterfaceType{
		Methods: &ast.FieldList{},
	}

	for i := 0; i < len(msgs); i++ {
		msg := &msgs[i]
		funcType := &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent("string")},
				},
			},
		}

		for i := 0; i < len(msg.Arguments); i++ {
			funcType.Params.List = append(funcType.Params.List, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(msg.Arguments[i].Name)},
				Type:  getPackageFieldType(&msg.Arguments[i]),
			})
		}

		ifaceType.Methods.List = append(ifaceType.Methods.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(msg.Name)},
			Type:  funcType,
		})
	}

	return ifaceType
}

func generateGeneralSupported(locs []Localization, decls *[]ast.Decl) {
	sliceLit := &ast.CompositeLit{
		Type: &ast.ArrayType{
			Elt: ast.NewIdent("string"),
		},
	}

	supportedSpec := &ast.ValueSpec{
		Names:  []*ast.Ident{ast.NewIdent("Supported")},
		Values: []ast.Expr{sliceLit},
	}

	varDecl := &ast.GenDecl{
		Tok:   token.VAR,
		Specs: []ast.Spec{supportedSpec},
	}

	for i := 0; i < len(locs); i++ {
		sliceLit.Elts = append(sliceLit.Elts, &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(locs[i].Lang.String()),
		})
	}

	*decls = append(*decls, varDecl)
}

func generateGeneralTable(locs []Localization, decls *[]ast.Decl) {
	mapLit := &ast.CompositeLit{
		Type: &ast.MapType{
			Key:   ast.NewIdent("string"),
			Value: ast.NewIdent("Localizer"),
		},
	}

	tableSpec := &ast.ValueSpec{
		Names:  []*ast.Ident{ast.NewIdent("mapLangToLocalizer")},
		Values: []ast.Expr{mapLit},
	}

	varDecl := &ast.GenDecl{
		Tok:   token.VAR,
		Specs: []ast.Spec{tableSpec},
	}

	for i := 0; i < len(locs); i++ {
		mapLit.Elts = append(mapLit.Elts, &ast.KeyValueExpr{
			Key: ast.NewIdent(strconv.Quote(locs[i].Lang.String())),
			Value: &ast.CompositeLit{
				Type: ast.NewIdent(getLocalizerTypeName(&locs[i])),
			},
		})
	}

	*decls = append(*decls, varDecl)
}

func generateGeneralFuncs(locs []Localization, decls *[]ast.Decl) {
	generateGeneralFuncNew(locs, decls)
	generateGeneralFuncLang(locs, decls)
}

func generateGeneralFuncNew(_ []Localization, decls *[]ast.Decl) {
	*decls = append(*decls, &ast.FuncDecl{
		Name: ast.NewIdent("New"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("lang")},
						Type:  ast.NewIdent("string"),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("loc")},
						Type:  ast.NewIdent("Localizer"),
					},
					{
						Names: []*ast.Ident{ast.NewIdent("ok")},
						Type:  ast.NewIdent("bool"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("loc"), ast.NewIdent("ok")},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{
						&ast.IndexExpr{
							X:     ast.NewIdent("mapLangToLocalizer"),
							Index: ast.NewIdent("lang"),
						},
					},
				},
				&ast.ReturnStmt{
					Results: []ast.Expr{
						ast.NewIdent("loc"),
						ast.NewIdent("ok"),
					},
				},
			},
		},
	})
}

func generateGeneralFuncLang(locs []Localization, decls *[]ast.Decl) {
	switchStmt := &ast.TypeSwitchStmt{
		Assign: &ast.ExprStmt{
			X: &ast.TypeAssertExpr{
				X: ast.NewIdent("loc"),
			},
		},
		Body: &ast.BlockStmt{},
	}

	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent("Language"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("loc")},
						Type:  ast.NewIdent("Localizer"),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent("string")},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{switchStmt},
		},
	}

	for i := 0; i < len(locs); i++ {
		switchStmt.Body.List = append(switchStmt.Body.List, &ast.CaseClause{
			List: []ast.Expr{
				ast.NewIdent(getLocalizerTypeName(&locs[i])),
			},
			Body: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: strconv.Quote(locs[i].Lang.String()),
						},
					},
				},
			},
		})
	}

	switchStmt.Body.List = append(switchStmt.Body.List, &ast.CaseClause{
		Body: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: `""`,
					},
				},
			},
		},
	})

	*decls = append(*decls, funcDecl)
}

func generateMessages(loc *Localization) (file *ast.File) {
	file = &ast.File{
		Name:  ast.NewIdent(config.PackageName),
		Decls: []ast.Decl{},
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// Code generated by go-l10n; DO NOT EDIT."},
				{Text: ""},
			},
		},
	}

	var decls []ast.Decl

	for i := 0; i < len(loc.Scopes); i++ {
		scope := &loc.Scopes[i]
		if scope.IsSimple() {
			generateSimpleMessage(loc, scope, &decls)
		} else {
			generateMessage(loc, scope, &decls)
		}
	}

	generateMessagesImportDecl(loc, &file.Decls)
	generateMessagesTypeDecl(loc, &file.Decls)

	file.Decls = append(file.Decls, decls...)

	return file
}

func generateMessagesImportDecl(loc *Localization, decls *[]ast.Decl) {
	importDecl := &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: []ast.Spec{},
	}

	for _, imp := range loc.Imports {
		importDecl.Specs = append(importDecl.Specs, &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(imp.Import),
			},
		})
	}

	if len(importDecl.Specs) != 0 {
		*decls = append(*decls, importDecl)
	}
}

func generateMessagesTypeDecl(loc *Localization, decls *[]ast.Decl) {
	typeDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(getLocalizerTypeName(loc)),
				Type: &ast.StructType{
					Fields: &ast.FieldList{},
				},
			},
		},
	}

	*decls = append(*decls, typeDecl)
}

func generateMessage(loc *Localization, scope *MessageScope, decls *[]ast.Decl) {
	const builderName = "builder"

	loc.AddImport(GoImport{"strings", "strings"})

	builderSpec := &ast.ValueSpec{
		Names: []*ast.Ident{
			ast.NewIdent(builderName),
		},
		Type: &ast.SelectorExpr{
			X:   ast.NewIdent("strings"),
			Sel: ast.NewIdent("Builder"),
		},
	}

	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent(getMessageFuncName(scope)),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent(getLocalizerName(loc))},
					Type:  ast.NewIdent(getLocalizerTypeName(loc)),
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent("string")},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok:   token.VAR,
						Specs: []ast.Spec{builderSpec},
					},
				},
			},
		},
	}

	for i := 0; i < len(scope.Arguments); i++ {
		funcDecl.Type.Params.List = append(funcDecl.Type.Params.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(scope.Arguments[i].Name)},
			Type:  getPackageFieldType(&scope.Arguments[i]),
		})
	}

	generators := []Generator{&scope.Plural, scope.String}

	for _, gen := range generators {
		if !gen.IsZero() {
			gen.Generate(loc, scope, builderName, &funcDecl.Body.List)
			break
		}
	}

	funcDecl.Body.List = append(funcDecl.Body.List, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent(builderName),
					Sel: ast.NewIdent("String"),
				},
			},
		},
	})

	for i := 0; i < len(scope.Variables); i++ {
		generateVariableFunc(loc, scope, &scope.Variables[i], decls)
	}

	*decls = append(*decls, funcDecl)
}

func generateSimpleMessage(loc *Localization, scope *MessageScope, decls *[]ast.Decl) {
	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent(getMessageFuncName(scope)),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent(getLocalizerName(loc))},
					Type:  ast.NewIdent(getLocalizerTypeName(loc)),
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent("string")},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{},
		},
	}

	generators := []Generator{&scope.Plural, scope.String}

	for _, gen := range generators {
		if !gen.IsZero() {
			gen.SimpleGenerate(loc, scope, &funcDecl.Body.List)
			break
		}
	}

	*decls = append(*decls, funcDecl)
}

func generatePlural(loc *Localization, scope *MessageScope, plural *Plural, builderName string, list *[]ast.Stmt) {
	generators := []struct {
		Gen   Generator
		Op    token.Token
		Value string
	}{
		{plural.Zero, token.EQL, "0"},
		{plural.One, token.EQL, "1"},
		{plural.Many, token.GTR, "1"},
		{plural.Other, token.ILLEGAL, ""},
	}

	switchStmt := &ast.SwitchStmt{
		Body: &ast.BlockStmt{},
	}

	for _, gen := range generators {
		if gen.Gen.IsZero() {
			continue
		}

		caseClause := &ast.CaseClause{}

		if gen.Op != token.ILLEGAL {
			caseClause.List = []ast.Expr{
				&ast.BinaryExpr{
					X:  ast.NewIdent(plural.Arg),
					Op: gen.Op,
					Y: &ast.BasicLit{
						Kind:  token.INT,
						Value: gen.Value,
					},
				},
			}
		}

		gen.Gen.Generate(loc, scope, builderName, &caseClause.Body)
		switchStmt.Body.List = append(switchStmt.Body.List, caseClause)
	}

	*list = append(*list, switchStmt)
}

func generateSimplePlural(loc *Localization, scope *MessageScope, plural *Plural, list *[]ast.Stmt) {
	generators := []struct {
		Gen   Generator
		Op    token.Token
		Value string
	}{
		{plural.Zero, token.EQL, "0"},
		{plural.One, token.EQL, "1"},
		{plural.Many, token.GTR, "1"},
		{plural.Other, token.ILLEGAL, ""},
	}

	switchStmt := &ast.SwitchStmt{
		Body: &ast.BlockStmt{},
	}

	for _, gen := range generators {
		if gen.Gen.IsZero() {
			continue
		}

		caseClause := &ast.CaseClause{}

		if gen.Op != token.ILLEGAL {
			caseClause.List = []ast.Expr{
				&ast.BinaryExpr{
					X:  ast.NewIdent(plural.Arg),
					Op: gen.Op,
					Y: &ast.BasicLit{
						Kind:  token.INT,
						Value: gen.Value,
					},
				},
			}
		}

		gen.Gen.SimpleGenerate(loc, scope, &caseClause.Body)
		switchStmt.Body.List = append(switchStmt.Body.List, caseClause)
	}

	*list = append(*list, switchStmt)
}

func (p *Plural) Generate(loc *Localization, scope *MessageScope, builderName string, list *[]ast.Stmt) {
	generatePlural(loc, scope, p, builderName, list)
}

func (p *Plural) SimpleGenerate(loc *Localization, scope *MessageScope, list *[]ast.Stmt) {
	generateSimplePlural(loc, scope, p, list)
}

func generateFormatParts(
	loc *Localization,
	scope *MessageScope,
	parts FormatParts,
	builderName string,
	list *[]ast.Stmt,
) {
	for _, part := range parts {
		switch part := part.(type) {
		case string:
			generateString(loc, scope, part, builderName, list)
		case ArgInfo:
			idx := argumentIndex(scope.Arguments, part.Name)
			generateArgument(loc, scope, &scope.Arguments[idx], &part, builderName, list)
		case VarInfo:
			idx := variableScopeIndex(scope.Variables, part.Name)
			generateVariableCall(loc, scope, &scope.Variables[idx], builderName, list)
		}
	}
}

func generateSimpleFormatParts(_ *Localization, _ *MessageScope, parts FormatParts, list *[]ast.Stmt) {
	var b strings.Builder

	for _, part := range parts {
		if part, ok := part.(string); ok {
			b.WriteString(part)
		}
	}

	*list = append(*list, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(b.String()),
			},
		},
	})
}

func (f FormatParts) Generate(loc *Localization, scope *MessageScope, builderName string, list *[]ast.Stmt) {
	generateFormatParts(loc, scope, f, builderName, list)
}

func (f FormatParts) SimpleGenerate(loc *Localization, scope *MessageScope, list *[]ast.Stmt) {
	generateSimpleFormatParts(loc, scope, f, list)
}

func generateString(_ *Localization, _ *MessageScope, str, builderName string, list *[]ast.Stmt) {
	*list = append(*list, &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(builderName),
				Sel: ast.NewIdent("WriteString"),
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(str),
				},
			},
		},
	})
}

func generateArgument(
	loc *Localization,
	_ *MessageScope,
	arg *Argument,
	info *ArgInfo,
	builderName string,
	list *[]ast.Stmt,
) {
	callExpr := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(builderName),
			Sel: ast.NewIdent("WriteString"),
		},
	}

	*list = append(*list, &ast.ExprStmt{
		X: callExpr,
	})

	if !info.FmtInfo.IsZero() {
		generateArgumentSprintf(loc, arg, info, callExpr)
		return
	}

	switch arg.GoType.Type {
	case "string":
		callExpr.Args = []ast.Expr{ast.NewIdent(arg.Name)}
	case "int":
		generateArgumentItoa(loc, arg, callExpr)
	case "float64":
		generateArgumentFormatFloat(loc, arg, callExpr)
	default:
		generateArgumentSprint(loc, arg, callExpr)
	}
}

func generateArgumentSprintf(loc *Localization, arg *Argument, info *ArgInfo, callExpr *ast.CallExpr) {
	fmtStr := info.FmtInfo.GoFormat(arg.GoType)
	loc.AddImport(GoImport{"fmt", "fmt"})

	callExpr.Args = []ast.Expr{
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("fmt"),
				Sel: ast.NewIdent("Sprintf"),
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(fmtStr),
				},
				ast.NewIdent(arg.Name),
			},
		},
	}
}

func generateArgumentItoa(loc *Localization, arg *Argument, callExpr *ast.CallExpr) {
	loc.AddImport(GoImport{"strconv", "strconv"})
	callExpr.Args = []ast.Expr{
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("strconv"),
				Sel: ast.NewIdent("Itoa"),
			},
			Args: []ast.Expr{ast.NewIdent(arg.Name)},
		},
	}
}

func generateArgumentFormatFloat(loc *Localization, arg *Argument, callExpr *ast.CallExpr) {
	loc.AddImport(GoImport{"strconv", "strconv"})
	callExpr.Args = []ast.Expr{
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("strconv"),
				Sel: ast.NewIdent("FormatFloat"),
			},
			Args: []ast.Expr{
				ast.NewIdent(arg.Name),
				&ast.BasicLit{
					Kind:  token.CHAR,
					Value: `'f'`,
				},
				&ast.BasicLit{
					Kind:  token.INT,
					Value: `6`,
				},
				&ast.BasicLit{
					Kind:  token.INT,
					Value: `64`,
				},
			},
		},
	}
}

func generateArgumentSprint(loc *Localization, arg *Argument, callExpr *ast.CallExpr) {
	loc.AddImport(GoImport{"fmt", "fmt"})
	callExpr.Args = []ast.Expr{
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("fmt"),
				Sel: ast.NewIdent("Sprint"),
			},
			Args: []ast.Expr{ast.NewIdent(arg.Name)},
		},
	}
}

func generateVariableCall(
	loc *Localization,
	scope *MessageScope,
	variable *VariableScope,
	builderName string,
	list *[]ast.Stmt,
) {
	callExpr := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(getLocalizerName(loc)),
			Sel: ast.NewIdent(getVariableFuncName(scope, variable)),
		},
		Args: []ast.Expr{
			&ast.UnaryExpr{
				Op: token.AND,
				X:  ast.NewIdent(builderName),
			},
		},
	}

	for _, name := range variable.ArgumentNames {
		callExpr.Args = append(callExpr.Args, ast.NewIdent(name))
	}

	*list = append(*list, &ast.ExprStmt{
		X: callExpr,
	})
}

func generateVariableFunc(loc *Localization, scope *MessageScope, variable *VariableScope, decls *[]ast.Decl) {
	const builderName = "b0"

	builderField := &ast.Field{
		Names: []*ast.Ident{
			ast.NewIdent(builderName),
		},
		Type: &ast.StarExpr{
			X: &ast.SelectorExpr{
				X:   ast.NewIdent("strings"),
				Sel: ast.NewIdent("Builder"),
			},
		},
	}

	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent(getVariableFuncName(scope, variable)),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent(getLocalizerName(loc))},
					Type:  ast.NewIdent(getLocalizerTypeName(loc)),
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{builderField},
			},
		},
		Body: &ast.BlockStmt{},
	}

	generators := []Generator{&variable.Plural, variable.String}

	for _, gen := range generators {
		if gen.IsZero() {
			continue
		}

		for _, name := range variable.ArgumentNames {
			argIdx := argumentIndex(scope.Arguments, name)
			funcDecl.Type.Params.List = append(funcDecl.Type.Params.List, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(name)},
				Type:  getPackageFieldType(&scope.Arguments[argIdx]),
			})
		}

		gen.Generate(loc, scope, builderName, &funcDecl.Body.List)
	}

	*decls = append(*decls, funcDecl)
}

func getPackageFieldType(arg *Argument) ast.Expr {
	if arg.GoType.Package == "" {
		return ast.NewIdent(arg.GoType.Type)
	}

	return &ast.SelectorExpr{
		X:   ast.NewIdent(arg.GoType.Package),
		Sel: ast.NewIdent(arg.GoType.Type),
	}
}

func getLocalizerName(loc *Localization) string {
	return loc.Lang.String() + "_l"
}

func getLocalizerTypeName(loc *Localization) string {
	return loc.Lang.String() + "_Localizer"
}

func getMessageFuncName(scope *MessageScope) string {
	return scope.Name
}

func getVariableFuncName(scope *MessageScope, variable *VariableScope) string {
	return scope.Name + "_" + variable.Name
}

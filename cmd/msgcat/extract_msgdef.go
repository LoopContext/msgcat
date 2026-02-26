package main

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/loopcontext/msgcat"
)

// isMessageDefType reports whether typ is msgcat.MessageDef or *msgcat.MessageDef from the given package.
func (e *keyExtractor) isMessageDefType(typ ast.Expr) bool {
	var sel *ast.SelectorExpr
	switch t := typ.(type) {
	case *ast.SelectorExpr:
		sel = t
	case *ast.StarExpr:
		sel, _ = t.X.(*ast.SelectorExpr)
	}
	if sel == nil {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return id.Name == e.msgcatName && sel.Sel.Name == "MessageDef"
}

// extractMessageDefFromCompositeLit extracts a MessageDef from a composite literal if possible.
// Returns key and RawMessage. key is empty if not a MessageDef or Key is missing.
func (e *keyExtractor) extractMessageDefFromCompositeLit(cl *ast.CompositeLit) (key string, raw msgcat.RawMessage) {
	if !e.isMessageDefType(cl.Type) {
		return "", msgcat.RawMessage{}
	}
	data := make(map[string]interface{})
	for _, elt := range cl.Elts {
		kve, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		kname, ok := kve.Key.(*ast.Ident)
		if !ok {
			continue
		}
		data[kname.Name] = e.exprToValue(kve.Value)
	}
	keyVal, _ := data["Key"]
	key, _ = keyVal.(string)
	if key == "" {
		return "", msgcat.RawMessage{}
	}
	raw.ShortTpl, _ = data["Short"].(string)
	raw.LongTpl, _ = data["Long"].(string)
	raw.PluralParam, _ = data["PluralParam"].(string)
	if raw.PluralParam == "" {
		raw.PluralParam = "count"
	}
	if sf, ok := data["ShortForms"].(map[string]string); ok && len(sf) > 0 {
		raw.ShortForms = sf
	}
	if lf, ok := data["LongForms"].(map[string]string); ok && len(lf) > 0 {
		raw.LongForms = lf
	}
	if c, ok := data["Code"]; ok {
		raw.Code = codeFromValue(c)
	}
	return key, raw
}

func (e *keyExtractor) exprToValue(expr ast.Expr) interface{} {
	switch t := expr.(type) {
	case *ast.BasicLit:
		if t.Kind == token.STRING {
			s, _ := unquote(t.Value)
			return s
		}
		if t.Kind == token.INT {
			n, _ := strconv.Atoi(t.Value)
			return n
		}
	case *ast.CompositeLit:
		// map literal e.g. map[string]string{"one": "...", "other": "..."}
		if m, ok := e.compositeLitToMap(t); ok {
			return m
		}
	}
	return nil
}

func (e *keyExtractor) compositeLitToMap(cl *ast.CompositeLit) (map[string]string, bool) {
	out := make(map[string]string)
	for _, elt := range cl.Elts {
		kve, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		k := e.exprToValue(kve.Key)
		v := e.exprToValue(kve.Value)
		ks, ok1 := k.(string)
		vs, ok2 := v.(string)
		if ok1 && ok2 {
			out[ks] = vs
		}
	}
	return out, len(out) > 0
}

func codeFromValue(v interface{}) msgcat.OptionalCode {
	switch t := v.(type) {
	case string:
		return msgcat.OptionalCode(t)
	case int:
		return msgcat.CodeInt(t)
	}
	return ""
}
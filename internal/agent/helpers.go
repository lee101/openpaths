package agent

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
)

func strArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func decodeB64(s string) ([]byte, error) {
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(s[:i], "base64") {
		s = s[i+1:]
	}
	return base64.StdEncoding.DecodeString(s)
}

// isReadOnlySQL allows a single SELECT/WITH statement and rejects writes.
func isReadOnlySQL(q string) bool {
	q = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(q), ";"))
	if q == "" || strings.Contains(q, ";") {
		return false
	}
	lower := strings.ToLower(q)
	if !strings.HasPrefix(lower, "select") && !strings.HasPrefix(lower, "with") {
		return false
	}
	for _, bad := range []string{"insert ", "update ", "delete ", "drop ", "alter ", "create ", "truncate ", "grant ", "revoke ", "copy ", "merge "} {
		if strings.Contains(lower, bad) {
			return false
		}
	}
	return true
}

func rowsToText(rows *sql.Rows, maxRows int) (string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(strings.Join(cols, " | ") + "\n")
	n := 0
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "", err
		}
		cells := make([]string, len(cols))
		for i, v := range vals {
			cells[i] = cellString(v)
		}
		b.WriteString(strings.Join(cells, " | ") + "\n")
		if n++; n >= maxRows {
			b.WriteString(fmt.Sprintf("... (truncated at %d rows)\n", maxRows))
			break
		}
	}
	if n == 0 {
		b.WriteString("(no rows)\n")
	}
	return b.String(), rows.Err()
}

func cellString(v any) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

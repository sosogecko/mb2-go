package web

import (
	"mb2-go/internal/util"
)

func (a *App) QueryMaps(sql string, args ...any) ([]map[string]any, error) {
	rows, err := a.DB.Raw(sql, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := make(map[string]any, len(cols))
		for i, col := range cols {
			m[col] = util.NormalizeRow(map[string]any{col: vals[i]})[col]
		}
		list = append(list, m)
	}
	if list == nil {
		list = []map[string]any{}
	}
	return list, rows.Err()
}

func (a *App) QueryMap(sql string, args ...any) (map[string]any, error) {
	list, err := a.QueryMaps(sql, args...)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// backend\internal\application\query\mall\catalog\catalog_query_util.go
package catalogQuery

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

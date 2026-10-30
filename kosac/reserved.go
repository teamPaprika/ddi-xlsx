package kosac

// defaultReservedWords are PostgreSQL's reserved keywords, both kinds: plain
// reserved and "reserved (can be function or type)". Neither is accepted as
// an unquoted column name. The warehouse DBMS is not settled yet; a different
// one replaces this list, or RefData.ReservedWords overrides it per call.
var defaultReservedWords = []string{
	"all", "analyse", "analyze", "and", "any", "array", "as", "asc", "asymmetric", "both", "case", "cast", "check",
	"collate", "column", "constraint", "create", "current_catalog", "current_date", "current_role", "current_time",
	"current_timestamp", "current_user", "default", "deferrable", "desc", "distinct", "do", "else", "end", "except",
	"false", "fetch", "for", "foreign", "from", "grant", "group", "having", "in", "initially", "intersect", "into",
	"lateral", "leading", "limit", "localtime", "localtimestamp", "not", "null", "offset", "on", "only", "or", "order",
	"placing", "primary", "references", "returning", "select", "session_user", "some", "symmetric", "system_user",
	"table", "then", "to", "trailing", "true", "union", "unique", "user", "using", "variadic", "when", "where",
	"window", "with",
	"authorization", "binary", "collation", "concurrently", "cross", "current_schema", "freeze", "full", "ilike",
	"inner", "is", "isnull", "join", "left", "like", "natural", "notnull", "outer", "overlaps", "right", "similar",
	"tablesample", "verbose",
}

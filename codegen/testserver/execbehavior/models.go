package execbehavior

// ValueViewer holds the root type by value, unlike the generated Viewer.
type ValueViewer struct {
	Name  string
	Query Query
}

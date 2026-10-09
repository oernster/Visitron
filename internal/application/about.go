package application

// Credit is one open source work shipped inside Visitron (FR-072).
type Credit struct {
	Work    string
	Licence string
	Holder  string
}

// Credits lists every open source work Visitron ships, Go modules and page
// packages alike. A structural test fails when go.mod or package.json gains a
// runtime dependency missing from here.
var Credits = []Credit{
	{Work: "Go and golang.org/x/sys", Licence: "BSD-3-Clause", Holder: "The Go Authors"},
	{Work: "Wails", Licence: "MIT", Holder: "Lea Anthony"},
	{Work: "modernc.org/sqlite", Licence: "BSD-3-Clause", Holder: "The Sqlite Authors"},
	{Work: "SQLite", Licence: "Public domain", Holder: "D. Richard Hipp and contributors"},
	{Work: "go-keyring", Licence: "MIT", Holder: "Zalando SE"},
	{Work: "React", Licence: "MIT", Holder: "Meta Platforms, Inc. and affiliates"},
	{Work: "GoatCounter (the service read)", Licence: "EUPL-1.2, slightly modified", Holder: "Martin Tournoij"},
}

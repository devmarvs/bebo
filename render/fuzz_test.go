package render

import "testing"

func FuzzTemplateNameFS(f *testing.F) {
	f.Add("templates", "templates/home.html")
	f.Add(".", "home.html")
	f.Add("templates", "templates/partials/_header.html")

	f.Fuzz(func(t *testing.T, dir, file string) {
		_, _ = templateNameFS(dir, file)
	})
}

func FuzzMatchPattern(f *testing.F) {
	f.Add("partials/*.html", "partials/_header.html")
	f.Add("**/*.html", "pages/home.html")

	f.Fuzz(func(t *testing.T, pattern, name string) {
		_ = matchPattern(pattern, name)
	})
}

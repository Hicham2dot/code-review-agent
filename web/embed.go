package web

import "embed"

//go:embed templates/dashboard.html templates/login.html templates/dashboard_user.html templates/dashboard_admin.html
var FS embed.FS

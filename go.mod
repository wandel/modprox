module github.com/wandel/modprox

go 1.18

require (
	github.com/go-git/go-billy/v5 v5.3.1
	github.com/go-git/go-git/v5 v5.4.2
	github.com/gorilla/mux v1.8.0
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli v1.22.9
	github.com/xanzy/go-gitlab v0.66.0
	golang.org/x/exp v0.0.0-20220518171630-0b5c67f07fdf
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3
)

require (
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0-20190314233015-f79a8a8ca69d // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.1 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/xanzy/ssh-agent v0.3.1 // indirect
	golang.org/x/crypto v0.0.0-20220513210258-46612604a0f9 // indirect
	golang.org/x/net v0.0.0-20220520000938-2e3eb7b945c2 // indirect
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

// some repositories fail due to https://github.com/go-git/go-git/pull/485
// remove this once this has been fixed
replace github.com/go-git/go-git/v5 => github.com/ZauberNerd/go-git/v5 v5.4.3-0.20220315170230-29ec1bc1e5db

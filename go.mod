module github.com/open-uem/openuem-worker

go 1.23.4

replace github.com/open-uem/openuem_ent => ./internal/ent

replace github.com/open-uem/openuem_utils => ./internal/utils

replace github.com/open-uem/openuem_nats => ./internal/nats

require (
	entgo.io/ent v0.14.1
	github.com/a-h/templ v0.2.793
	github.com/open-uem/openuem_ent v0.0.0-00010101000000-000000000000
	github.com/open-uem/openuem_nats v0.0.0-00010101000000-000000000000
	github.com/open-uem/openuem_utils v0.0.0-00010101000000-000000000000
	github.com/go-co-op/gocron/v2 v2.14.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/nats-io/nats.go v1.38.0
	github.com/urfave/cli/v2 v2.27.5
	github.com/wneessen/go-mail v0.5.2
	golang.org/x/crypto v0.31.0
	golang.org/x/sys v0.28.0
	gopkg.in/ini.v1 v1.67.0
	software.sslmate.com/src/go-pkcs12 v0.5.0
)

require (
	ariga.io/atlas v0.29.1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/go-openapi/inflect v0.21.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/hcl/v2 v2.23.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jonboulle/clockwork v0.4.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/zclconf/go-cty v1.15.1 // indirect
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67 // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.28.0 // indirect
)

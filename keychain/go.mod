module github.com/jtamagnan/git-utils/keychain

go 1.21

require (
	github.com/jtamagnan/git-utils/keychain/lib v0.0.0
	github.com/spf13/cobra v1.8.0
	golang.org/x/term v0.15.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.15.0 // indirect
)

replace github.com/jtamagnan/git-utils/keychain/lib => ./lib

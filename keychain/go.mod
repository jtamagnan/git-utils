module github.com/jtamagnan/git-utils/keychain

go 1.21

require (
	github.com/jtamagnan/git-utils/keychain/lib v0.0.0
	golang.org/x/term v0.15.0
)

require golang.org/x/sys v0.15.0 // indirect

replace github.com/jtamagnan/git-utils/keychain/lib => ./lib

# Prints `make help` from the Makefile's "target: ... ## description" lines.
# Runs inside the go container, so the host needs no grep or awk.
BEGIN {
	FS = ":.*## "
	print "Usage: make <target>\n"
}
/^[a-zA-Z_-]+:.*## / {
	printf "  \033[36m%-16s\033[0m %s\n", $1, $2
}

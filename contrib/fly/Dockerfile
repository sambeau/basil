# Basil on a Fly Machine.
#
# There is no build stage: ./build.sh cross-compiles Basil to a static Linux
# binary and drops it next to this file, so the image is a copy. Read the note
# in build.sh about `-tags nodynamic` before changing how that binary is made —
# CGO_ENABLED=0 on its own is not enough to make it run here.
#
# The base image is not `scratch`, and cannot be, for two reasons:
#
#   git   Basil shells out to it. The Git endpoint runs git-upload-pack and
#         git-receive-pack (server/git.go), deploy materialises a release with
#         `git archive` (server/deploy/release.go), and the release branch is
#         read with `git symbolic-ref` (server/deploy/repo.go).
#   sh    The receive hooks Basil installs are two-line /bin/sh scripts
#         (server/deploy/hooks.go).
#
# ca-certificates is for ACME and for outbound calls from handlers; tzdata is
# for date formatting in Parsley.

FROM alpine:3.22

RUN apk add --no-cache git ca-certificates tzdata

# /usr/local/bin is deliberate and worth keeping stable. The receive hooks bake
# in the *absolute* path of the binary that installed them, so a path that moves
# between image builds leaves hooks pointing at nothing. Basil rewrites drifted
# hooks at startup, so this self-heals — but not moving is better than healing.
COPY basil /usr/local/bin/basil
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

# 80 answers the ACME challenge, 443 serves the site.
EXPOSE 80 443

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

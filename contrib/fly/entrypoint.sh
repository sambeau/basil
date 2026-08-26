#!/bin/sh
# Start Basil against the site root on the mounted volume.
#
# The one thing that cannot be baked into the image is `basil --init`: it
# writes the site root, and the site root lives on a Fly Volume that does not
# exist until the Machine boots. So the first boot has nothing to serve.
#
# Rather than initialise automatically, this waits. `basil --init --server`
# prints the admin API key exactly once and it is not recoverable — printing it
# into `fly logs` is the wrong place for it. Run the init by hand, once, over
# `fly ssh console`, where you are looking at the output.

set -e

SITE="${BASIL_SITE:-/srv/mysite}"

if [ ! -d "$SITE/site.git" ]; then
	echo "basil: $SITE has no site.git — the site root has not been initialised."
	echo "basil:"
	echo "basil: Run this once, from your laptop:"
	echo "basil:"
	echo "basil:   fly ssh console -C \"basil --init $SITE --server --host YOUR.HOSTNAME --admin YOUR-NAME\""
	echo "basil:"
	echo "basil: Write down the API key it prints — it is shown once. Then:"
	echo "basil:"
	echo "basil:   fly machine restart"
	echo "basil:"
	echo "basil: Holding the Machine open so you can ssh in."
	# Not `sleep infinity`: this runs under BusyBox on Alpine, whose sleep
	# takes a number and nothing else.
	while :; do sleep 3600; done
fi

exec basil --site "$SITE"

if [ "$1" = "--build" ]; then
    ./tools/build.sh
fi

if [ "$1" = "-d" -o "$2" = "-d" ]; then
    # background
    docker-compose up -d
    echo
    echo "[SUCCESS] app running as docker-compose on background!"
    echo "(To terminate app: \"docker-compose down\")"
    echo
else
    # foreground
    docker-compose up
fi

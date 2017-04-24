set -ex

./build.sh
docker push pedronasser/functions-lambda:nodejs4.3

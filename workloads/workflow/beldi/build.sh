#!/bin/bash
set -euo pipefail

BUILD_TYPE="Release"
while [ ! $# -eq 0 ]
do
  case "$1" in
    --debug)
      BUILD_TYPE="Debug"
      ;;
  esac
  shift
done

PROJECT_DIR="$(realpath $(dirname "$0")/../../..)"
WORKFLOW_DIR=$PROJECT_DIR/workloads/workflow
BOKI_DIR=$PROJECT_DIR/boki

export CGO_ENABLED=1
CGO_CFLAGS="$(go env CGO_CFLAGS) -I$BOKI_DIR/lib/shared_index/include"
if [[ $BUILD_TYPE == "Debug" ]]; then
    CGO_LDFLAGS="$(go env CGO_LDFLAGS) -L$BOKI_DIR/lib/shared_index/bin/debug -lrt -ldl -lindex"
else
    CGO_LDFLAGS="$(go env CGO_LDFLAGS) -L$BOKI_DIR/lib/shared_index/bin/release -lrt -ldl -lindex"
fi

function build_singleop {
	go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/singleop/init internal/singleop/init/init.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=1000" -o bin/singleop/singleop internal/singleop/main/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=1000" -o bin/singleop/nop internal/singleop/nop/nop.go

	# CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI -X main.TXN=ENABLE -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=1000" -o bin/tsingleop/tsingleop internal/singleop/main/main.go
	# CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI -X main.TXN=ENABLE -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=1000" -o bin/tsingleop/tnop internal/singleop/nop/nop.go

	go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bsingleop/init internal/singleop/init/init.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bsingleop/singleop internal/singleop/main/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bsingleop/nop internal/singleop/nop/nop.go
}

function build_hotel_baseline {
	go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/init internal/hotel/init/init.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/geo internal/hotel/main/handlers/geo/geo.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/profile internal/hotel/main/handlers/profile/profile.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/rate internal/hotel/main/handlers/rate/rate.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/recommendation internal/hotel/main/handlers/recommendation/recommendation.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/user internal/hotel/main/handlers/user/user.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/search internal/hotel/main/handlers/search/search.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/hotel internal/hotel/main/handlers/hotel/hotel.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/flight internal/hotel/main/handlers/flight/flight.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/order internal/hotel/main/handlers/order/order.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/frontend internal/hotel/main/handlers/frontend/frontend.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bhotel/gateway internal/hotel/main/handlers/gateway/gateway.go
}

function build_hotel {
	go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/init internal/hotel/init/init.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/geo internal/hotel/main/handlers/geo/geo.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/profile internal/hotel/main/handlers/profile/profile.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/rate internal/hotel/main/handlers/rate/rate.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/recommendation internal/hotel/main/handlers/recommendation/recommendation.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/user internal/hotel/main/handlers/user/user.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/search internal/hotel/main/handlers/search/search.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/hotel internal/hotel/main/handlers/hotel/hotel.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/flight internal/hotel/main/handlers/flight/flight.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/order internal/hotel/main/handlers/order/order.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/frontend internal/hotel/main/handlers/frontend/frontend.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/gateway internal/hotel/main/handlers/gateway/gateway.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/gc internal/hotel/main/gc/gc.go
	# CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/hotel/collector internal/hotel/main/collector/collector.go
}

function build_media_baseline {
	go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/init internal/media/init/init.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/CastInfo internal/media/core/handlers/castInfo/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/ComposeReview internal/media/core/handlers/composeReview/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/Frontend internal/media/core/handlers/frontend/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/MovieId internal/media/core/handlers/movieId/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/MovieInfo internal/media/core/handlers/movieInfo/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/MovieReview internal/media/core/handlers/movieReview/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/Page internal/media/core/handlers/page/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/Plot internal/media/core/handlers/plot/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/Rating internal/media/core/handlers/rating/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/ReviewStorage internal/media/core/handlers/reviewStorage/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/Text internal/media/core/handlers/text/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/UniqueId internal/media/core/handlers/uniqueId/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/User internal/media/core/handlers/user/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE" -o bin/bmedia/UserReview internal/media/core/handlers/userReview/main.go
}

function build_media {
	go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/init internal/media/init/init.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/CastInfo internal/media/core/handlers/castInfo/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/ComposeReview internal/media/core/handlers/composeReview/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/Frontend internal/media/core/handlers/frontend/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/MovieId internal/media/core/handlers/movieId/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/MovieInfo internal/media/core/handlers/movieInfo/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/MovieReview internal/media/core/handlers/movieReview/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/Page internal/media/core/handlers/page/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/Plot internal/media/core/handlers/plot/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/Rating internal/media/core/handlers/rating/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/ReviewStorage internal/media/core/handlers/reviewStorage/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/Text internal/media/core/handlers/text/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/UniqueId internal/media/core/handlers/uniqueId/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/User internal/media/core/handlers/user/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/UserReview internal/media/core/handlers/userReview/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/gc internal/media/core/gc/gc.go
	# CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/media/collector internal/media/core/collector/collector.go
}

function build_gctest {
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/gctest/gctest internal/gctest/core/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/gctest/gc internal/gctest/core/gc/gc.go
}

function build_gctesttxn {
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=101" -o bin/gctest/gctest internal/gctest/core/main.go
	CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=101" -o bin/gctest/gc internal/gctest/core/gc/gc.go
}

function clean {
	rm -rf ./bin
}

build_hotel_baseline
build_media_baseline
build_hotel
build_media
build_singleop

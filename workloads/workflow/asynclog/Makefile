# COMPILE_FLAGS = -race
COMPILE_FLAGS = 

LDFLAGS_BASELINE = -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BASELINE \
				   -X github.com/eniac/Beldi/pkg/cayonlib.TYPE=BASELINE
LDFLAGS_BELDI = -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI \
				-X github.com/eniac/Beldi/pkg/cayonlib.TYPE=BELDI
LDFLAGS_COMMON = -s -w

.PHONY: build clean deploy

build:
# single operation

all: singleop hotel media finra

singleop: singleop_init singleop_beldi_main singleop_beldi_nop \
	singleop_baseline_init singleop_baseline_main singleop_baseline_nop
singleop_init:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/singleop/init internal/singleop/init/init.go
singleop_beldi_main:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI) -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=1000" -o bin/singleop/singleop internal/singleop/main/main.go
singleop_beldi_nop:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI) -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=1000" -o bin/singleop/nop internal/singleop/nop/nop.go
# singleop_beldi_tmain:
# 	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI) -X main.TXN=ENABLE -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=1000" -o bin/tsingleop/tsingleop internal/singleop/main/main.go
# singleop_beldi_tnop:
# 	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI) -X main.TXN=ENABLE -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=1000" -o bin/tsingleop/tnop internal/singleop/nop/nop.go
singleop_baseline_init:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bsingleop/init internal/singleop/init/init.go
singleop_baseline_main:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bsingleop/bsingleop internal/singleop/main/main.go
singleop_baseline_nop:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bsingleop/bnop internal/singleop/nop/nop.go

hotel-baseline: hotel_baseline_init hotel_baseline_geo hotel_baseline_profile \
	hotel_baseline_rate hotel_baseline_recommendation hotel_baseline_user \
	hotel_baseline_search hotel_baseline_hotel hotel_baseline_flight hotel_baseline_order \
	hotel_baseline_frontend hotel_baseline_gateway
hotel_baseline_init:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/init internal/hotel-baseline/init/init.go
hotel_baseline_geo:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/geo internal/hotel-baseline/main/handlers/geo/geo.go
hotel_baseline_profile:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/profile internal/hotel-baseline/main/handlers/profile/profile.go
hotel_baseline_rate:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/rate internal/hotel-baseline/main/handlers/rate/rate.go
hotel_baseline_recommendation:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/recommendation internal/hotel-baseline/main/handlers/recommendation/recommendation.go
hotel_baseline_user:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/user internal/hotel-baseline/main/handlers/user/user.go
hotel_baseline_search:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/search internal/hotel-baseline/main/handlers/search/search.go
hotel_baseline_hotel:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/hotel internal/hotel-baseline/main/handlers/hotel/hotel.go
hotel_baseline_flight:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/flight internal/hotel-baseline/main/handlers/flight/flight.go
hotel_baseline_order:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/order internal/hotel-baseline/main/handlers/order/order.go
hotel_baseline_frontend:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/frontend internal/hotel-baseline/main/handlers/frontend/frontend.go
hotel_baseline_gateway:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bhotel/gateway internal/hotel-baseline/main/handlers/gateway/gateway.go

hotel: hotel_init hotel_geo hotel_profile hotel_rate hotel_recommendation hotel_user \
	hotel_search hotel_hotel hotel_flight hotel_order hotel_frontend hotel_gateway hotel_gc
hotel_init:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/init internal/hotel/init/init.go
hotel_geo:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/geo internal/hotel/main/handlers/geo/geo.go
hotel_profile:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/profile internal/hotel/main/handlers/profile/profile.go
hotel_rate:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/rate internal/hotel/main/handlers/rate/rate.go
hotel_recommendation:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/recommendation internal/hotel/main/handlers/recommendation/recommendation.go
hotel_user:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/user internal/hotel/main/handlers/user/user.go
hotel_search:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/search internal/hotel/main/handlers/search/search.go
hotel_hotel:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/hotel internal/hotel/main/handlers/hotel/hotel.go
hotel_flight:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/flight internal/hotel/main/handlers/flight/flight.go
hotel_order:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/order internal/hotel/main/handlers/order/order.go
hotel_frontend:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/frontend internal/hotel/main/handlers/frontend/frontend.go
hotel_gateway:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/gateway internal/hotel/main/handlers/gateway/gateway.go
hotel_gc:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/gc internal/hotel/main/gc/gc.go
# hotel_collector:
# 	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/hotel/collector internal/hotel/main/collector/collector.go

media-baseline: media_baseline_init media_baseline_CastInfo media_baseline_ComposeReview \
	media_baseline_Frontend media_baseline_MovieId media_baseline_MovieInfo \
	media_baseline_MovieReview media_baseline_Page media_baseline_Plot media_baseline_Rating \
	media_baseline_ReviewStorage media_baseline_Text media_baseline_UniqueId media_baseline_User media_baseline_UserReview
media_baseline_init:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/init internal/media-baseline/init/init.go
media_baseline_CastInfo:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/CastInfo internal/media-baseline/core/handlers/castInfo/main.go
media_baseline_ComposeReview:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/ComposeReview internal/media-baseline/core/handlers/composeReview/main.go
media_baseline_Frontend:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/Frontend internal/media-baseline/core/handlers/frontend/main.go
media_baseline_MovieId:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/MovieId internal/media-baseline/core/handlers/movieId/main.go
media_baseline_MovieInfo:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/MovieInfo internal/media-baseline/core/handlers/movieInfo/main.go
media_baseline_MovieReview:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/MovieReview internal/media-baseline/core/handlers/movieReview/main.go
media_baseline_Page:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/Page internal/media-baseline/core/handlers/page/main.go
media_baseline_Plot:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/Plot internal/media-baseline/core/handlers/plot/main.go
media_baseline_Rating:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/Rating internal/media-baseline/core/handlers/rating/main.go
media_baseline_ReviewStorage:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/ReviewStorage internal/media-baseline/core/handlers/reviewStorage/main.go
media_baseline_Text:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/Text internal/media-baseline/core/handlers/text/main.go
media_baseline_UniqueId:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/UniqueId internal/media-baseline/core/handlers/uniqueId/main.go
media_baseline_User:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/User internal/media-baseline/core/handlers/user/main.go
media_baseline_UserReview:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BASELINE)" -o bin/bmedia/UserReview internal/media-baseline/core/handlers/userReview/main.go

media: media_init media_CastInfo media_ComposeReview media_Frontend media_MovieId \
	media_MovieInfo media_MovieReview media_Page media_Plot media_Rating \
	media_ReviewStorage media_Text media_UniqueId media_User media_UserReview media_Gc
media_init:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/init internal/media/init/init.go
media_CastInfo:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/CastInfo internal/media/core/handlers/castInfo/main.go
media_ComposeReview:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/ComposeReview internal/media/core/handlers/composeReview/main.go
media_Frontend:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/Frontend internal/media/core/handlers/frontend/main.go
media_MovieId:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/MovieId internal/media/core/handlers/movieId/main.go
media_MovieInfo:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/MovieInfo internal/media/core/handlers/movieInfo/main.go
media_MovieReview:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/MovieReview internal/media/core/handlers/movieReview/main.go
media_Page:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/Page internal/media/core/handlers/page/main.go
media_Plot:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/Plot internal/media/core/handlers/plot/main.go
media_Rating:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/Rating internal/media/core/handlers/rating/main.go
media_ReviewStorage:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/ReviewStorage internal/media/core/handlers/reviewStorage/main.go
media_Text:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/Text internal/media/core/handlers/text/main.go
media_UniqueId:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/UniqueId internal/media/core/handlers/uniqueId/main.go
media_User:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/User internal/media/core/handlers/user/main.go
media_UserReview:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/UserReview internal/media/core/handlers/userReview/main.go
media_Gc:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/gc internal/media/core/gc/gc.go
# media_Collector:
# 	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/media/collector internal/media/core/collector/collector.go

finra: fetchData lastpx marginBalance marketdata side trddate volume
fetchData:
	env GOOS=linux go build -buildvcs=false -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/finra/fetchData internal/finra/cmd/fetchData/main.go
lastpx:
	env GOOS=linux go build -buildvcs=false -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/finra/lastpx internal/finra/cmd/lastpx/main.go
marginBalance:
	env GOOS=linux go build -buildvcs=false -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/finra/marginBalance internal/finra/cmd/marginBalance/main.go
marketdata:
	env GOOS=linux go build -buildvcs=false -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/finra/marketdata internal/finra/cmd/marketdata/main.go
side:
	env GOOS=linux go build -buildvcs=false -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/finra/side internal/finra/cmd/side/main.go
trddate:
	env GOOS=linux go build -buildvcs=false -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/finra/trddate internal/finra/cmd/trddate/main.go
volume:
	env GOOS=linux go build -buildvcs=false -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/finra/volume internal/finra/cmd/volume/main.go

gctest:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/gctest/gctest internal/gctest/core/main.go
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI)" -o bin/gctest/gc internal/gctest/core/gc/gc.go

gctesttxn:
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI) -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=101" -o bin/gctest/gctest internal/gctest/core/main.go
	env GOOS=linux go build -buildvcs=false $(COMPILE_FLAGS) -ldflags="$(LDFLAGS_COMMON) $(LDFLAGS_BELDI) -X github.com/eniac/Beldi/pkg/beldilib.DLOGSIZE=101" -o bin/gctest/gc internal/gctest/core/gc/gc.go

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose

.PHONY: build test package
test:
	go test ./navcore -count=1

# Supply the two SDL include directories via CGO_CFLAGS.
build:
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 \
	CC="zig cc -target arm-linux-gnueabihf.2.17 -mcpu=cortex_a7" \
	CGO_LDFLAGS="-Wl,--allow-shlib-undefined" \
	go build -trimpath -ldflags="-s -w" -tags netgo,osusergo -o App/MiyooDrome/MiyooDrome ./src

package: build
	sh scripts/package.sh

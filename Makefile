.PHONY: build build-test test package package-test
test:
	go test ./navcore -count=1

# Supply the two SDL include directories via CGO_CFLAGS.
build: player_output = App/MiyooDrome/MiyooDrome
build-test: player_output = dist/test-bin/MiyooDrome
build-test: edition_tags = ,testapp
build-test: edition_flags = $(if $(BUILD_ID),-X 'main.buildSuffix= ($(BUILD_ID))')

build build-test:
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 \
	CC="zig cc -target arm-linux-gnueabihf.2.17 -mcpu=cortex_a7" \
	CGO_LDFLAGS="-Wl,--allow-shlib-undefined" \
	go build -trimpath -ldflags="-s -w $(edition_flags)" -tags netgo,osusergo$(edition_tags) -o $(player_output) ./src

package: build
	sh scripts/package.sh

package-test: build-test
	sh scripts/package.sh test

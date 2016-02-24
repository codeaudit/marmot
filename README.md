# Marmot Image Checker
This simple app was dreamt up after coming across [this TechCrunch article](http://techcrunch.com/2016/02/18/google-opens-its-cloud-vision-api-to-all-developers/). It take an image, asks the Google Cloud Vision API for three descriptions of the image, and compares those descriptions to a chosen list of words. If there is match, the image is added to the [toadserver](https://github.com/eris-ltd/toadserver). This is a WIP, experimental, and probably full of bugs.

## Dependencies
### Tools
- `go`
- `docker`
- `eris`

### Google Cloud Vision API
- Get setup with your API key on the [Google Cloud Platform](https://cloud.google.com/vision/docs/getting-started)
- Dig Deeper into the [API](https://cloud.google.com/vision/docs/concepts) to tweak some of the default settings.

### Environment Variables
```
export CLOUD_VISION_API_KEY=browser_key
export CLOUD_VISION_MARMOT_CHECKS="rodent,groundhog,marmot,squirrel"
export TOADSERVER_HOST=$(eris services inspect toadserver_srv NetworkSettings.IPAddress)
```
where `browser_key` is got from Google, and the second env var is a list of words to check the image description against.

The last one should be set after `run.sh` (see below) and is used to link to the toadserver running as a service to the marmot checker. This is normally abstracted away via servicification with the `eris` tool for which an example is forthcoming.

## Install & Run
Install repo on your `$GOPATH`
```
go get github.com/eris-ltd/marmot
```

`cd` into the repo and run `bash run.sh`

This'll setup a single validator chain with keys sorted, and the toadserver started alongside IPFS. Kill the script once the toadserver has started. (Now set `TOADSERVER_HOST`) Then:

```
go run main.go
```
to start the marmot checker.

## Check An Image
From another screen (or host):
```
curl -X POST http://localhost:2332/postImage/marmot.png --data-binary "@marmot.png"
```
where `marmot.png` is an image in your `pwd` that you'd like to know if it is indeed, a marmot (or any descriptor listed in `CLOUD_VISION_MARMOT_CHECKS`). If it is, in fact, a marmot, then it will be added to the linked toadserver.

See [this tutorial](https://docs.erisindustries.com/tutorials/advanced/servicesmaking/) for more information on checking that it was added.

## With Docker
When your `pwd` is this repo and assuming `run.sh` has been run:
```
docker build -t quay.io/eris/marmot .
docker run -d -p 2332:2332 --link eris_service_toadserver_srv_1:ts -e "TOADSERVER_HOST=ts" -e "CLOUD_VISION_API_KEY=$CLOUD_VISION_API_KEY" -e "CLOUD_VISION_MARMOT_CHECKS=$CLOUD_VISION_MARMOT_CHECKS" quay.io/eris/marmot
```
Then *Check An Image*.

## With an Eris Service
```
Tomorrow
```

## Code Examples
Coming Soon!

## Why
This would be useful, say, to archive and index digital content that only meets certain parameters. I imagine a future where budding school-aged scientists will submit images of insects they've found out in the field, alongside a geo-tag, to a chain that aggregates insect populations.

## TODO
- Dockerfile
- service-ify
- integration tests
- unit tests
- sane flexibility for "features"

## Contributions
Always welcome. Or fork and run with it.

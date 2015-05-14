# istore

istore is to fill the space between storage, database, application and image recognition.
This component is

- an object storage similar to S3
- an image proxy that caches contents as well as processes pixels
- a name service that provides uniform access to different data sources
- a hierarchical database system that maintains metadata associated with each image data, and provides prefix-key iteration
- a backend storage system to applications as well as various database products

## Dependency

At the time of wrting, istore depends on ffmpeg installed on the system with pkg-config.
In the latest Ubuntu, there is no official package for ffmpeg anymore, so you should build
it.  Refer to https://gist.github.com/xdamman/e4f713c8cd1a389a5917

## User Guide

Currently there is no special client module.  You can interact with istore using curl.

### Simple Operation

#### POST

You can POST or PUT any URL object under a path.  `metadata` parameter can register a json
associated with the path key.

```
$ curl -XPOST $HOST/path/sample/http://video.webmfiles.org/elephants-dream.webm -d metadata='{"name": "my video"}'
```

PUT overwrites the metadata entirely with the input json, whereas POST method merges the input
with the existing json.

#### GET

After you register an object, you can query it.

```
$ curl -XGET $HOST/path/sample/http://video.webmfiles.org/elephants-dream.webm
```

This will return the object at the original URL.  istore caches the object.

#### LIST

If you GET at the directory, istore returns the list of json under the directory.


```
$ curl -XGET $HOST/path/sample/

[{"_id":493,"_filepath":"/path/sample/http://video.webmfiles.org/elephants-dream.webm","metadata":{"name":"my video"}}]
```

### Image Processing

istore implements most of the image processing from the imaging package.  To call each function,
simply add ?apply={function}&{param}={value}... to GET request.

- adjustBrightness(percentage)
- adjustContrast(percentage)
- adjustGamma(gamma)
- adjustSigmoid(midpoint, factor)
- blur(sigma)
- crop(x1, y1, x2, y2)
- drawRect(rects=[(x1, y1, x2, y2, r, g, b)...])
- fit(w, h)
- flipH()
- flipV()
- grayscale()
- invert()
- sharpen(sigmoid)
- transpose()
- transverse()
- resize(w, h)

For video objects, the below function is available.

- frame(sec)

See also https://godoc.org/github.com/disintegration/imaging


### Video Slicing

You can slice video into frames by

```
$ curl -XPOST $HOST/path/slice/_expand -d '{"video": "/path/to/video"}'
```

It registers as many objects as duration of the video.

### URL Scheme

Currently the following URL schemes is handled.

- http, https
  Retrieves object from remote http(s)
- file
  Retrieves object from the local disk of istore
- self
  Retrieves object from the istore path.  This makes it possible to nested image processing.

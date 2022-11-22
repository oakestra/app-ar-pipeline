# Client

Client application for the AR pipeline.
The client supports: 
- 2 entrypoint (main and backup)
- configurable fps and buffer size
- configurable bounding boxes frequency request

# Endpoints exposed
The client exposes and endpoint at ``/api/result``
and expects teh following json format encoding the Bounding Boxes:
```
{
  "frame-id":int
  "bb":[
     {
       "x1":int,
       "x2":int,
       "y1":int,
       "y2":int,
       "label":string
     }
   ]
}
```

# Run

First use the following command to update the dependencies

```
go get -u
```

Then run the application using 

```
go run main.go
```

You can even build the executables with go build.

Use `go run main.go -h` to get check the supported flags

# Flags

Usage of main:
```
  -backupentry string
        backup cloud entrypoint for the pipeline (default "0.0.0.0:5000")
  -bbps int
        bounding boxes per second to ber requested (default 3)
  -buffer int
        frame buffer size (default 30)
  -entry string
        entrypoint for the pipeline (default "0.0.0.0:5000")
  -fps int
        set the frames per second captured by the came (default 30)
  -port int
        bounding boxes per second to ber requested (default 40100)

```
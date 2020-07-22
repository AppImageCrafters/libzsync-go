# libzsync-go

libzsync implementation in Golang

See http://zsync.moria.org.uk/

## Usage

```go
import github.com/AppImageCrafters/libzsync-go
```

```go
// Configure
sync, _ := zsync.NewZSync("https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage.zsync")
sync.RemoteFileUrl = "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"

// Execute
output, _ := os.Create("/tmp/appimagetool-new-x86_64.AppImage")
err = sync.Sync("/tmp/appimagetool-x86_64.AppImage", output)
```


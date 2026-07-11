package transport

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

func writeBinaryDownload(
	w http.ResponseWriter,
	traceID string,
	download bridgeorchestration.BinaryDownload,
	disposition string,
	includeChecksum bool,
) {
	if traceID != "" {
		w.Header().Set("X-Trace-ID", traceID)
	}
	writeBinaryDownloadHeaders(w, download, disposition, includeChecksum)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, download.Reader)
}

func writeBinaryDownloadHeaders(
	w http.ResponseWriter,
	download bridgeorchestration.BinaryDownload,
	disposition string,
	includeChecksum bool,
) {
	w.Header().Set("Content-Type", download.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", download.Size))
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, download.Name))
	if includeChecksum && strings.TrimSpace(download.SHA256) != "" {
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, download.SHA256))
		w.Header().Set("X-Artifact-SHA256", download.SHA256)
	}
}

package function

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	vision "cloud.google.com/go/vision/apiv1"
	visionpb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

// GCSEvent is the payload of a GCS Event
type GCSEvent struct {
	Kind                    string                 `json:"kind"`
	ID                      string                 `json:"id"`
	SelfLink                string                 `json:"selfLink"`
	Name                    string                 `json:"name"`
	Bucket                  string                 `json:"bucket"`
	Generation              string                 `json:"generation"`
	Metageneration          string                 `json:"metageneration"`
	ContentType             string                 `json:"contentType"`
	TimeCreated             time.Time              `json:"timeCreated"`
	Updated                 time.Time              `json:"updated"`
	TemporaryHold           bool                   `json:"temporaryHold"`
	EventBasedHold          bool                   `json:"eventBasedHold"`
	RetentionExpirationTime time.Time              `json:"retentionExpirationTime"`
	StorageClass            string                 `json:"storageClass"`
	TimeStorageClassUpdated time.Time              `json:"timeStorageClassUpdated"`
	Size                    string                 `json:"size"`
	MD5Hash                 string                 `json:"md5Hash"`
	MediaLink               string                 `json:"mediaLink"`
	ContentEncoding         string                 `json:"contentEncoding"`
	ContentDisposition      string                 `json:"contentDisposition"`
	CacheControl            string                 `json:"cacheControl"`
	Metadata                map[string]interface{} `json:"metadata"`
	CRC32C                  string                 `json:"crc32c"`
	ComponentCount          int                    `json:"componentCount"`
	Etag                    string                 `json:"etag"`
	CustomerEncryption      struct {
		EncryptionAlgorithm string `json:"encryptionAlgorithm"`
		KeySha256           string `json:"keySha256"`
	}
	KMSKeyName    string `json:"kmsKeyName"`
	ResourceState string `json:"resourceState"`
}

var (
	visionClient  *vision.ImageAnnotatorClient
	storageClient *storage.Client

	resultBucket string
)

func setup(ctx context.Context) error {
	resultBucket = os.Getenv("RESULT_BUCKET")

	var err error
	if visionClient == nil {
		visionClient, err = vision.NewImageAnnotatorClient(ctx)
		if err != nil {
			return fmt.Errorf("vision.NewImageAnnotatorClient: %v", err)
		}
	}
	if storageClient == nil {
		storageClient, err = storage.NewClient(ctx)

		if err != nil {
			return fmt.Errorf("storageClient creation failure: %v", err)
		}
	}

	return nil
}

//PdfToCloudVision is a Cloud Function Handler
func PdfToCloudVision(ctx context.Context, e GCSEvent) error {
	log.Printf("Bucket: %v\n", e.Bucket)
	log.Printf("File: %v\n", e.Name)

	if strings.HasSuffix(e.Name, ".pdf") {
		if err := setup(ctx); err != nil {
			return fmt.Errorf("Setup Error: %v", err)
		}

		if resultBucket == "" {
			return fmt.Errorf(("RESULT_BUCKET is empty"))
		}

		log.Printf("Searching text in %v", e.Name)

		request := &visionpb.BatchAnnotateFilesRequest{
			Requests: []*visionpb.AnnotateFileRequest{
				{
					Features: []*visionpb.Feature{
						{Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION},
					},
					InputConfig: &visionpb.InputConfig{
						GcsSource: &visionpb.GcsSource{Uri: fmt.Sprintf("gs://%s/%s", e.Bucket, e.Name)},
						MimeType:  "application/pdf",
					},
				},
			},
		}

		res, err := visionClient.BatchAnnotateFiles(ctx, request)

		if err != nil {
			return fmt.Errorf("Detext texts faild: %v", err)
		}

		firstResp := res.Responses[0]

		resultFileName := strings.Replace(e.Name, ".pdf", ".txt", 1)
		fullTextBuilder := strings.Builder{}

		for _, v := range firstResp.Responses {
			fullTextBuilder.WriteString(v.FullTextAnnotation.Text)
		}

		writer := storageClient.Bucket(resultBucket).Object(resultFileName).NewWriter(ctx)
		writer.ContentType = "text/plain; charset=\"utf-8\""

		defer writer.Close()

		if _, err := writer.Write([]byte(fullTextBuilder.String())); err != nil {
			return fmt.Errorf("Writing Result faild: %v", err)
		}
	}

	return nil
}

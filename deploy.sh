#!/bin/bash

gcloud functions deploy pdf-ocr-func --runtime go113 --trigger-bucket ${TRIGGER_BUCKET} --entry-point PdfToCloudVision --set-env-vars "^:^RESULT_BUCKET=${RESULT_BUCKET}"
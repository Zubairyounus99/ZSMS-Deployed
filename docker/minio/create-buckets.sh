#!/bin/sh
# Wait for MinIO service to respond
until (/usr/bin/mc config host add zsmsminio http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD"); do
  echo "Waiting for MinIO..."
  sleep 1
done

# Create default media bucket
/usr/bin/mc mb --ignore-existing zsmsminio/zsms-media
/usr/bin/mc anonymous set download zsmsminio/zsms-media || true
echo "MinIO zsms-media bucket configured successfully."
exit 0

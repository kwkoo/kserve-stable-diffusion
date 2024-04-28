#!/usr/bin/env python3

import requests
import boto3
import os
import pathlib
import typing

# extracts basename from path and strips off extension
def model_name_from_path(p: str) -> str:
    return pathlib.Path(p).stem

def lines_from_file(p: str) -> typing.List[str]:
    if not os.path.isfile(p):
        raise Exception(f'file {p} does not exist')

    with open(p, "r") as f:
        return [line.rstrip('\n') for line in f]

def filename_from_url(url: str) -> str:
    sep = url.rfind('?')
    if sep != -1:
        url = url[:sep]
    sep = url.rfind('/')
    if sep != -1:
        url = url[sep+1:]
    return url

def log(msg: str) -> None:
    print(msg, flush=True)

class Client:

    def __init__(self, access_key_id, access_secret, s3_endpoint):
        if access_key_id is None or access_key_id == '':
            raise Exception('access_key_id is not set')
        if access_secret is None or access_secret == '':
            raise Exception('access_secret is not set')
        if s3_endpoint is None or s3_endpoint == '':
            raise Exception('s3_endpoint is not set')
        session = boto3.session.Session()
        self.s3_client = session.client(
            service_name='s3',
            aws_access_key_id=access_key_id,
            aws_secret_access_key=access_secret,
            endpoint_url=s3_endpoint
        )

    def upload_from_url_to_bucket(self, url: str, bucket: str, model_name: str) -> None:
        r = requests.get(url, stream=True)
        self.s3_client.upload_fileobj(r.raw, bucket, model_name + '/' + filename_from_url(url))

    def download_from_bucket(self, bucket: str) -> None:
        resp = self.s3_client.list_objects_v2(Bucket=bucket)
        if resp is None:
            log('could not get response')
            return
        contents = resp.get('Contents')
        if contents is None:
            log('bucket had no contents')
            return
        for entry in contents:
            key = entry.get('Key')
            if key is None:
                continue
            dir = pathlib.Path(key).parent
            if not dir.is_dir():
                # directory does not exist - we need to create it ahead of time
                dir.mkdir(parents=True, exist_ok=True)
            log(f'downloading {key}')
            self.s3_client.download_file(bucket, key, key)

    def upload_model_to_bucket(self, bucket, model_name, urls) -> None:
        if bucket is None or bucket == '':
            raise Exception('bucket is not set')
        for url in urls:
            url = url.strip()
            if len(url) == 0:
                continue
            log(f'uploading {url}')
            self.upload_from_url_to_bucket(url, bucket, model_name)


if __name__ == '__main__':
    client = Client(
        os.environ.get('AWS_ACCESS_KEY_ID'),
        os.environ.get('AWS_SECRET_ACCESS_KEY'),
        os.environ.get('AWS_ENDPOINT_URL_S3')
    )
    model_paths = os.environ.get('MODELS')
    if model_paths is None:
        raise Exception('MODELS environment variable is not set')
    
    for model_path in model_paths.split(','):
        model_path = model_path.strip()
        model_name = model_name_from_path(model_path)
        if model_name == '':
            log('could not extract model name from path')
            continue
        log(f'model name = {model_name}')
        model_urls = lines_from_file(model_path)
        if len(model_urls) == 0:
            log(f'could not get urls from {model_path}')
            continue
        client.upload_model_to_bucket(os.environ.get('S3_BUCKET', 'models'), model_name, model_urls)

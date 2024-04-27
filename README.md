# Text to Image Demo

## Preparing the model archive (`.mar`)

We will use [`OFA-Sys / small-stable-diffusion-v0`](https://huggingface.co/OFA-Sys/small-stable-diffusion-v0)

01. Ensure you have `git-lfs` setup, then clone the model directory

		git clone https://huggingface.co/OFA-Sys/small-stable-diffusion-v0

01. Prepare `model.zip`

		cd small-stable-diffusion-v0

		rm -rf .git

		zip -r ../model.zip .

01. Create `.mar`

		torch-model-archiver \
		  --model-name sd \
		  --version 1.0 \
		  --serialized-file model.zip \
		  --handler custom_handler.py \
		  -r requirements.txt


## Testing with `torchserve`

01. Start `torchserve`

		torchserve \
		  --start \
		  --model-store . \
		  --models sd=sd.mar \
		  --ts-config config.properties

01. Call the inference API

		curl \
		  -s \
		  -F 'data=an apple' \
		  localhost:8085/predictions/sd \
		| \
		base64 -d > apple.jpg


## Deploying on OpenShift

01. Provision an `AWS Blank Open Environment` in `ap-southeast-1`, create an OpenShift cluster with 2 `p3.2xlarge` worker nodes

	*   Create a new directory for the install files

			mkdir demo

			cd demo

	*   Generate `install-config.yaml`

			openshift-install create install-config

	*   Set the compute pool to 1 replica with a `p3.2xlarge` instance, and set the control plane to a single master (you will need to have `yq` installed)

			mv install-config.yaml install-config-old.yaml

			yq '.compute[0].replicas=1' < install-config-old.yaml \
			| \
			yq '.compute[0].platform = {"aws":{"zones":["ap-southeast-1b"], "type":"p3.2xlarge"}}' \
			| \
			yq '.controlPlane.replicas=1' \
			> install-config.yaml

	*   Create the cluster

			openshift-install create cluster
			
		You may get a `context deadline exceeded` error - this is expected because there is only a single control-plane node

01. Set the `KUBECONFIG` environment variable to point to the new cluster

01. Setup the ingress with certificates from Let's Encrypt

		./scripts/setup-letsencrypt
	
	Note: After the certificates have been installed, you will need to edit `kubeconfig` and comment out `.clusters[*].cluster.certificate-authority-data`

01. Deploy OpenShift AI and its dependencies to OpenShift

		make deploy-infra
	
	This will:

	*   Configure OpenShift for User Workload Monitoring
	*   Deploy the NFD and Nvidia GPU operators
	*   Deploy the OpenShift Serverless and Service Mesh operators
	*   Deploy OpenShift AI and KServe
	*   Deploy minio

01. Upload the `.mar` to minio

	*   Get the URL of minio

			make minio-console

	*   Open a web browser to minio and login as `minio` / `minio123`

	*   Create an S3 bucket named `models`

	*   Open the `models` bucket

		*   Create a folder named `config` and upload `config.properties`
		*   Create a folder named `model-store` and upload `sd.mar`

01. Deploy the `InferenceService`

		make deploy-sd

01. Send a test request to the `InferenceService`

		model="$(oc get inferenceservice/sd -n demo -o jsonpath='{.status.url}')"

		curl \
		  -sk \
		  $model/v2/models/sd/infer \
		  -H 'Content-Type: application/json' \
		  -d '
		{"inputs": [{
		  "name":"dummy",
		  "shape": [-1],
		  "datatype":"STRING",
		  "data":["an apple"]
		}]}' \
		| \
		jq -r '.outputs[0].data[0]' \
		| \
		base64 -d > apple.jpg

01. Deploy the frontend

		make deploy-frontend


## References

*   [waveglow handler](https://github.com/pytorch/serve/blob/master/examples/text_to_speech_synthesizer/waveglow_handler.py) - example of using zip files




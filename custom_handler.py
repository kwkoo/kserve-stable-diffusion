import logging
import os
import zipfile
import tempfile
import io
import torch
import base64
from diffusers import StableDiffusionPipeline
from ts.torch_handler.base_handler import BaseHandler


logger = logging.getLogger(__name__)

try:
    import torch_xla.core.xla_model as xm

    XLA_AVAILABLE = True
except ImportError as error:
    XLA_AVAILABLE = False


class StableDiffusionHandler(BaseHandler):

    def __init__(self):
        super(StableDiffusionHandler, self).__init__()
        self._context = None
        self.initialized = False


    def initialize(self, context):
        self._context = context

        # Set device type
        if torch.cuda.is_available():
            self.device = torch.device("cuda")
        elif torch.backends.mps is not None and torch.backends.mps.is_available():
            self.device = torch.device("mps")
        elif XLA_AVAILABLE:
            self.device = xm.xla_device()
        else:
            self.device = torch.device("cpu")

        logger.info("torch device = %s", self.device)

        # Load the model
        properties = context.system_properties
        self.manifest = context.manifest
        model_dir = properties.get("model_dir")
        if self.manifest["model"].get("serializedFile") is None:
            raise RuntimeError("serializedFile is not defined")

        serialized_file = self.manifest["model"]["serializedFile"]
        zip_path = os.path.join(model_dir, serialized_file)

        with tempfile.TemporaryDirectory(dir=model_dir) as expandedmodel:
            logger.info("extract %s to %s...", zip_path, expandedmodel)
            with zipfile.ZipFile(zip_path, "r") as zip_ref:
                zip_ref.extractall(expandedmodel)
            pipe = StableDiffusionPipeline.from_pretrained(expandedmodel, torch_dtype=torch.float16)

        self.model = pipe.to(self.device)
        logger.debug("model loaded successfully")
        self.initialized = True
    
    def preprocess(self, data):
        logger.info("input data = %s", data)
        return [str(el.get('data')) for el in data]
    
    def inference(self, model_input):
        logger.info("inference model input = %s", model_input)
        if model_input is None:
            raise RuntimeError('model input is None')

        return self.model(model_input).images

    def postprocess(self, res):
        logger.info('res = %s', res)
        output = []
        for result in res:
            bytes = io.BytesIO()
            result.save(bytes, format='JPEG')
            output.append(base64.b64encode(bytes.getvalue()).decode('ascii'))
        return output

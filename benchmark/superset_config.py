"""Benchmark-only Superset settings. Not a production configuration."""

import os

SECRET_KEY = os.environ["SUPERSET_SECRET_KEY"]

# The benchmark drives SQL Lab through the REST API with a bearer token on a
# private address, so CSRF is off here. Do not copy this to a real deployment.
WTF_CSRF_ENABLED = False

SQLLAB_TIMEOUT = 60
SUPERSET_WEBSERVER_TIMEOUT = 120

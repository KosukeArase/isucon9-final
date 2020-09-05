import multiprocessing


bind = "0.0.0.0:8000"
workers = multiprocessing.cpu_count() * 2 + 1
reload = True
worker_class = "gevent"
worker_connections = 1000

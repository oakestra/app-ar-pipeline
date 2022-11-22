#FROM nvcr.io/nvidia/l4t-pytorch:r32.6.1-pth1.9-py3 #this one is specific for l4t devices like the nvidia jetson
FROM python:3.8

ENV LC_CTYPE en_US.UTF-8
ENV LANG en_US.UTF-8

#RUN apt-get update \
#    && apt-get install --no-install-recommends -y \
#        language-pack-en \
#        python3-dev \
#        build-essential \
#        git \
#        cmake \
#        wget \
#        libz-dev \
#        libncurses-dev \
#        libbz2-dev \
#        liblzma-dev \
#    && pip3 --no-cache-dir install --upgrade \
#        setuptools \
#        wheel

RUN apt-get update
RUN apt-get install ffmpeg libsm6 libxext6 -y
#RUN apt-get install python-numpy
RUN apt-get clean
ADD requirements.txt .

RUN python3 -m pip install --upgrade pip
RUN pip3 install numpy
#RUN pip3 install --upgrade pip setuptools wheel
#RUN pip3 install opencv-python
RUN pip3 install -r requirements.txt
RUN pip3 install opencv-python

ADD . .

CMD ["python3","detection.py"]

#FROM mcr.microsoft.com/azureml/onnxruntime:v.1.4.0-jetpack4.4-l4t-base-r32.4.3
FROM mcr.microsoft.com/azureml/onnxruntime:latest

ENV LC_CTYPE en_US.UTF-8
ENV LANG en_US.UTF-8

RUN apt-get update
RUN apt-get install --no-install-recommends -y \
        language-pack-en \
        python3-dev \
        build-essential \
        git \
        cmake \
        wget \
        libz-dev \
        libncurses-dev \
        libbz2-dev \
        liblzma-dev

RUN wget https://go.dev/dl/go1.20.6.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.20.6.linux-amd64.tar.gz

RUN python3 -m pip install --upgrade pip
RUN apt install ffmpeg libsm6 libxext6 -y
RUN apt-get install python-numpy -y
RUN apt-get clean

RUN python3 -m pip install --upgrade pip
RUN pip3 uninstall -y protobuf
RUN pip3 install protobuf>=3.5.0
RUN apt install -y libprotobuf-dev
RUN apt install -y protobuf-compiler
RUN apt install -y libprotobuf-dev protobuf-compiler python3-scipy libprotoc-dev

RUN apt install -y git
RUN git clone https://github.com/pybind/pybind11.git
RUN pip3 install pytest
RUN cd pybind11 && git checkout tags/v2.2.0 && mkdir build && cd build && cmake .. && make install

RUN pip3 install onnx==1.4.1
RUN pip3 install Cython
RUN pip3 install opencv-python
RUN pip3 install -U insightface==0.4.1

ADD requirements.txt .
RUN pip3 install -r requirements.txt
#RUN pip3 uninstall onnxruntime
#RUN pip3 install onnxruntime-gpu==1.4.0
RUN apt install -y unzip

RUN cd /root && mkdir .insightface
ADD models /root/.insightface/models
RUN cd ~/.insightface/models && mkdir buffalo_sc && unzip buffalo_sc.zip -d buffalo_sc

#Install sidecar queue
ENV PATH=$PATH:/usr/local/go/bin
RUN wget https://github.com/giobart/Active-Internal-Queue/archive/refs/tags/v1.0.10.tar.gz
RUN tar -xf v1.0.10.tar.gz
WORKDIR Active-Internal-Queue-1.0.10/cmd/SidecarQueue
RUN go build -o bin/SidecarQueue sidecarQueue.go
RUN mv bin/SidecarQueue /SidecarQueue
WORKDIR /

RUN pip uninstall -y protobuf
RUN pip install protobuf==4.21.0
RUN cd ~/.insightface/models && mkdir buffalo_s && unzip buffalo_s.zip -d buffalo_s

#ENV OPENBLAS_CORETYPE=ARMV8
ADD . .
CMD ["python3","recognition.py"]

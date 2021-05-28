FROM elasticsearch:7.12.1
RUN yum install -y -q -e 0 unzip
COPY ./elasticsearch-expert-scripts.zip /usr/share/elasticsearch
COPY ./lib /usr/share/elasticsearch/lib
RUN echo "y" | elasticsearch-plugin install file:///usr/share/elasticsearch/elasticsearch-expert-scripts.zip

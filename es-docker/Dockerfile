FROM elasticsearch:7.12.1
RUN yum install -y -q -e 0 unzip
COPY ./elasticsearch-expert-scripts.zip /usr/share/elasticsearch
COPY ./lib /usr/share/elasticsearch/lib
COPY ./rest_handler_cat_example.zip /usr/share/elasticsearch
RUN echo "y" | elasticsearch-plugin install file:///usr/share/elasticsearch/elasticsearch-expert-scripts.zip
RUN echo "y" | elasticsearch-plugin install file:///usr/share/elasticsearch/rest_handler_cat_example.zip

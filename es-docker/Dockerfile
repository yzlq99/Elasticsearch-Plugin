FROM elasticsearch:7.12.1
RUN yum install -y -q -e 0 unzip
COPY ./elastic-search-sikp-list-plugin.zip /usr/share/elasticsearch
COPY ./lib /usr/share/elasticsearch/lib
RUN echo "y" | elasticsearch-plugin install file:///usr/share/elasticsearch/elastic-search-sikp-list-plugin.zip

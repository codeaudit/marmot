FROM quay.io/eris/base
MAINTAINER Eris Industries <support@erisindustries.com>

ENV NAME         marmot
ENV REPO 	 eris-ltd/$NAME
ENV BRANCH       master
ENV BINARY_PATH  $NAME
ENV CLONE_PATH   $GOPATH/src/github.com/$REPO
ENV INSTALL_PATH $INSTALL_BASE/$NAME

# for binary
ENV INSTALL_BASE /usr/local/bin

RUN mkdir -p $CLONE_PATH

#for local buildz
COPY . $CLONE_PATH
WORKDIR $CLONE_PATH


#RUN git clone -q https://github.com/$REPO $CLONE_PATH
#RUN git checkout -q $BRANCH
RUN cd $CLONE_PATH && go build -o $INSTALL_BASE/marmot

USER $USER
WORKDIR $ERIS

VOLUME $ERIS
EXPOSE 2332
#CMD ["marmot"]
ENTRYPOINT ["marmot"]

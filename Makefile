
CC=gcc
CFLAGS=-I$(HOME)/local/include -fPIC -g -O2

CXX=g++
CXXFLAGS=-I$(HOME)/local/include -fPIC -g -O2

LIBS=-Wl,-Bstatic -lboost_filesystem -lboost_system -Wl,-Bdynamic
LDFLAGS=-fPIC -L$(HOME)/local/lib

SHARED=libreplican.so
SHARED_OBJS=Blocks.o sha1.o

OBJS=$(SHARED_OBJS) Main.o

all: $(SHARED) replican

replican: Main.o $(SHARED)
	$(CXX) -o $@ $^ -Wl,-rpath=. -L. $(LDFLAGS) -lreplican $(LIBS)

$(SHARED): $(SHARED_OBJS)
	$(CXX) -shared -o $@ $(LDFLAGS) $(LIBS) $^

%.o:	%.cpp
	$(CXX) $(CXXFLAGS) -c -o $@ $^

%.o:	%.c
	$(CC) $(CFLAGS) -c -o $@ $^

clean:
	$(RM) -f $(OBJS)



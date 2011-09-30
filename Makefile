
CC=gcc
CFLAGS=-I$(HOME)/local/include -fPIC -g -O2

CXX=g++
CXXFLAGS=-I$(HOME)/local/include -fPIC -g -O2

LIBS=-Wl,-Bstatic -lboost_filesystem -lboost_system -Wl,-Bdynamic
LDFLAGS=-fPIC -L$(HOME)/local/lib

SHARED=libreplican.so
SHARED_OBJS=Blocks.o sha1.o

MAIN_OBJS=Main.o

TEST=libreplican_tests.so
TEST_OBJS=Test.o
TEST_LIBS=-Wl,-Bstatic -lboost_unit_test_framework -Wl,-Bdynamic $(LIBS)

OBJS=$(SHARED_OBJS) $(TEST_OBJS) $(MAIN_OBJS)

ARTIFACTS=$(SHARED) replican replitests

all: $(ARTIFACTS)

replican: $(MAIN_OBJS) $(SHARED)
	$(CXX) -o $@ $(MAIN_OBJS) -Wl,-rpath=. -L. $(LDFLAGS) -lreplican $(LIBS)

$(SHARED): $(SHARED_OBJS)
	$(CXX) -shared -o $@ $(LDFLAGS) $(LIBS) $^

replitests: $(TEST_OBJS) $(SHARED)
	$(CXX) -o $@ $(TEST_OBJS) -Wl,-rpath=. -L. $(LDFLAGS) -lreplican $(TEST_LIBS)
	
%.o:	%.cpp
	$(CXX) $(CXXFLAGS) -c -o $@ $^

%.o:	%.c
	$(CC) $(CFLAGS) -c -o $@ $^

clean:
	$(RM) $(OBJS) $(ARTIFACTS)



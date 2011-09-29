
#include "Blocks.hpp"
#include <iostream>

int main(int argc, char** argv) {
    if (argc < 2) {
        std::cerr << "Usage: <root path>" << std::endl;
        return 1;
    }
    
    boost::filesystem::path root(argv[1]);
    replican::index_dir(root);
}


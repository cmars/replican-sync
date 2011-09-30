
#include "Blocks.hpp"
#define BOOST_TEST_MODULE ReplicanTest
//#define BOOST_TEST_MAIN
#include <boost/test/unit_test.hpp>

#include <iostream>

using namespace replican;

BOOST_AUTO_TEST_CASE(test_declarative_tree) {
    
    DirPtr root(new Dir("root"));
    
    DirPtr etc(new Dir("etc"));
    root->add_child(etc);
    
    FilePtr etc_passwd(new File("passwd"));
    etc->add_child(etc_passwd);
    
    FilePtr etc_hosts(new File("hosts"));
    etc->add_child(etc_hosts);
    
    DirPtr usr(new Dir("usr"));
    root->add_child(usr);
    
    DirPtr usr_bin(new Dir("bin"));
    usr->add_child(usr_bin);
    
    DirPtr usr_lib(new Dir("lib"));
    usr->add_child(usr_bin);
    
    DirPtr usr_share(new Dir("share"));
    usr->add_child(usr_share);
    
    FilePtr usr_bin_ls(new File("ls"));
    usr_bin->add_child(usr_bin_ls);
    
    BOOST_CHECK_EQUAL(usr_bin_ls->get_path().string(), "root/usr/bin/ls");
    
    BOOST_CHECK_EQUAL(etc->get_children().size(), 2);
    
}

BOOST_AUTO_TEST_CASE(replican_test2) {
    
    std::cout << "hello world2" << std::endl;
    
    BOOST_CHECK(true);
}



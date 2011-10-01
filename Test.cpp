
#include "Blocks.hpp"
#define BOOST_TEST_MODULE ReplicanTest
//#define BOOST_TEST_MAIN
#include <boost/test/unit_test.hpp>

#include <boost/filesystem.hpp>

#include <iostream>

using namespace replican;
namespace fs = boost::filesystem;

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

BOOST_AUTO_TEST_CASE(file_hashing) {
    fs::path mp4file("./testroot/My Music/0 10k 30.mp4");
    
    FilePtr f = replican::index_file(mp4file);
    BOOST_CHECK_EQUAL(f->get_strong(), "5ab3e5d621402e5894429b5f595a1e2d7e1b3078");
    
    BlockPtr b = boost::static_pointer_cast<Block>(f->get_children()[0]);
    BOOST_CHECK_EQUAL(b->get_strong(), "d1f11a93449fa4d3f320234743204ce157bbf1f3");
    
    b = boost::static_pointer_cast<Block>(f->get_children()[1]);
    BOOST_CHECK_EQUAL(b->get_strong(), "eabbe570b21cd2c5101a18b51a3174807fa5c0da");
}

BOOST_AUTO_TEST_CASE(tree_hashing) {
    fs::path testroot_path("./testroot");
    DirPtr testroot = replican::index_dir(testroot_path);
    
    BOOST_CHECK_EQUAL(testroot->get_strong(), "ddf07ff332d0493de9ab208bf9a060fde4c8186");
}



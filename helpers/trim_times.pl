#!/usr/bin/perl -W
#

while (<>) {
  if ( $_ =~ /.*item_lists.* (.*) (.*)/) {
    print "item_list,$1,,,,$2\n"
  } elsif ($_ =~ /.*annotations.* (.*) (.*)/) {
    print "annotation,,$1,,,$2\n"
  } elsif ($_ =~ /.*doc.* (.*) (.*)/) {
    print "doc,,,$1,,$2\n"
  } elsif ($_ =~ /.*primary.* (.*) (.*)/) {
    print "doc,,,$1,,$2\n"
  } elsif ($_ =~ /.* (.*) (.*)/) {
    print "item,,,,$1,$2\n"
  }
}

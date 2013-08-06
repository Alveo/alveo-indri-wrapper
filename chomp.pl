#!/usr/bin/perl
#
#  Reads the output of run.sh and makes a csv table
#
#

my $i=0;

die "Please specify number of iterations" unless $#ARGV + 1 == 1; 

my $num_iterations = $ARGV[0];

for (my $k = 1 ; $k <= $num_iterations; $k++) {
  print "Iteration $k,";
}
print "\n";
                            
while(<STDIN>) {
  chomp;
  if($_ =~ /(\d+)m([\d\.]+)s/) {
    my $val = (60* $1 + $2);
    print "$val,";
    $i++;
    if($i% $num_iterations == 0) {
      print "\n";
    }
  } else {
    die "incorrectly formatted line $_"
  }
}

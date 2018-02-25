#!/usr/bin/perl

use strict;
use warnings;

my @suites = ("♤ ","♧ ","♡ ","♢ ");
my @values = (2 .. 9, "T", "Q", "K", "A");

my @stack;
initialize_stack(\@stack);

foreach (1 .. 2000000) {
  fisher_yates_shuffle(\@stack);

  if (($stack[0] =~ /A/) && ($stack[1] =~ /A/)) {
    print "Aces!\n";
  }
}

exit;

# from http://www.perlmonks.org/?node_id=1869
# randomly permutate @array in place
sub fisher_yates_shuffle {
  my $array = shift;
  my $i = @$array;
  while ( --$i ) {
    my $j = int rand( $i+1 );
    @$array[$i,$j] = @$array[$j,$i];
  }
}

sub initialize_stack {
  my $array = shift;

  foreach my $suit (@suites) {
    foreach my $value (@values) {
      push(@$array, "$value$suit");
    }
  }
}

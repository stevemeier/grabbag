#!/usr/bin/perl

use strict;
use warnings;
use Data::Dumper;
use List::Util qw(shuffle);

my @suites = ("♤ ","♧ ","♡ ","♢ ");
my @values = (2 .. 9,"T","J","Q","K","A");

my %pocket;

my @stack;
initialize_stack(\@stack);

foreach (1 .. 10000000) {
# fisher_yates_shuffle(\@stack);
  @stack = shuffle @stack;

  if (($stack[0] =~ /^A/) && ($stack[1] =~ /^A/)) { $pocket{'aces'}++; }
  if (($stack[0] =~ /^K/) && ($stack[1] =~ /^K/)) { $pocket{'kings'}++; }
  if (($stack[0] =~ /^Q/) && ($stack[1] =~ /^Q/)) { $pocket{'queens'}++; }
  if (($stack[0] =~ /^J/) && ($stack[1] =~ /^J/)) { $pocket{'jacks'}++; }
}

print Dumper %pocket;
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

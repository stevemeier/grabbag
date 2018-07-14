#!/usr/bin/perl

use strict;
use warnings;
use Data::Dumper;
use Getopt::Long;
use LWP::UserAgent qw(head);
use LWP::Protocol::https;

# Default values
my $warn = 3600;
my $crit = 86400;

# Construct
my $ua = LWP::UserAgent->new;
$ua->timeout(5);

my $head;
my $lastmod1;
my $lastmod2;
my $agediff;
my ($ref, $url);

GetOptions('w=s' => \$warn,
	   'warn=s' => \$warn,
	   'c=s' => \$crit,
	   'crit=s' => \$crit,
	   'r=s' => \$ref,
	   'ref=s' => \$ref,
	   'u=s' => \$url,
	   'url=s' => \$url);

if ($head = $ua->head($ref)) {
  if (defined($head->last_modified)) { 
    $lastmod1 = $head->last_modified;
  } else {
    print "UNKNOWN: No last modified date at $ref\n";
    exit 3;
  }
} else {
  print "UNKNOWN: Could not fetch $ref";
  exit 3;
}

if ($head = $ua->head($url)) {
  if (defined($head->last_modified)) {
    $lastmod2 = $head->last_modified;
  } else {
    print "UNKNOWN: No last modified date at $url\n";
    exit 3;
  }
} else {
  print "UNKNOWN: Could not fetch $url";
  exit 3;
}

if ((defined($lastmod1)) && (defined($lastmod2))) {
  $agediff = ($lastmod1 - $lastmod2);
  if ($agediff >= $crit) {
    print "CRITICAL: $url is out of date by $agediff seconds\n";
    exit 2 }
  elsif  ($agediff >= $warn) {
    print "WARNING: $url is out of date by $agediff seconds\n";
    exit 1;
  } else {
    print "OK: Difference is $agediff seconds\n";
    exit 0;
  }
} else {
  print "UNKNOWN: Could not compare timestamps";
  exit 3;
}

exit;


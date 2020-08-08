#!/usr/bin/perl

# Nagios-style check script to check `etckeeper`
# Alerts on non-commited changes in /etc
#
# Author: Steve Meier
# Date:   2020-08-08

use strict;
use warnings;
use Monitoring::Plugin;

my $binary = "/usr/bin/etckeeper";
my $modified = 0;
my $deleted = 0;
my $untracked = 0;

my $mp = Monitoring::Plugin->new(usage => "Usage: %s");

if (not(-x $binary)) {
  $mp->plugin_exit(UNKNOWN, "etckeeper binary ($binary) not found")
}

open(my $EKSTATUS, '-|', "$binary vcs status -s");
while(<$EKSTATUS>) {
  if (/^\sM/ix) { $modified++ }
  if (/^\sD/ix) { $deleted++ }
  if (/^\?\?/ix) { $untracked++ }
}
close($EKSTATUS);

if ($? > 0) {
  $mp->plugin_exit(UNKNOWN, "etckeeper exited with status ".($? >> 8))
}

if ( ($modified > 0) || ($deleted > 0) ) {
  $mp->plugin_exit(CRITICAL, "$modified file(s) modified, $deleted file(s) deleted")
}

if ($untracked > 0) {
  $mp->plugin_exit(WARNING, "$untracked untracked file(s)")
}

if ( ($modified eq 0) &&
     ($deleted eq 0) &&
     ($untracked eq 0) ) {
  $mp->plugin_exit(OK, "Everything is in order");
}

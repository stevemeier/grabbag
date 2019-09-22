#!/usr/bin/perl

use strict;
use warnings;
use File::Temp qw(:POSIX);
use Monitoring::Plugin;
use XML::Simple;

my $nmap = '/usr/bin/nmap';
my $nmapopts = "-n";
my $tempfile = tmpnam();

my $mp = Monitoring::Plugin->new(usage => "Usage: %s -H <host>");
$mp->add_arg(spec => 'host|H=s', help => 'Host', required => 1);
$mp->add_arg(spec => 'tcp|t=s',  help => 'TCP ports which should be open');
$mp->add_arg(spec => 'udp|u=s',  help => 'UDP ports which should be open');
$mp->add_arg(spec => 'bin|b=s',  help => 'Path of nmap binary to use');
$mp->add_arg(spec => 'opt|o=s',  help => 'Options to pass through to nmap');

# Parse arguments
$mp->getopts;

# Default or custom nmap path
if (defined($mp->opts->bin)) { $nmap = $mp->opts->bin; } 

# Pass through nmap options
if (defined($mp->opts->opt)) { $nmapopts = $mp->opts->opt; }

# TCP mode
if (defined($mp->opts->tcp)) { $nmapopts .= " -sT"; }

# UDP mode
if (defined($mp->opts->udp)) { $nmapopts .= " -sU"; }

# Limit port range to specified ports only
$nmapopts .= " -p ";
$nmapopts .= &port_range($mp->opts->tcp, $mp->opts->udp);

# Add output file
$nmapopts .= ' -oX '.$tempfile;

# Add host
$nmapopts .= " ".$mp->opts->host;

# Check nmap binary
if (not(-x $nmap)) {
  $mp->plugin_exit(UNKNOWN, "nmap binary not found or not executable at $nmap")
}

# Run nmap
system("$nmap $nmapopts >/dev/null");
if ($? > 0) {
  $mp->plugin_exit(UNKNOWN, "nmap exited with code $?");
}

# Read XML result and remove tempfile
my $nmapxml = XMLin($tempfile);
unlink($tempfile);

# Put ports to be checked into a more useful data structure
my %ports;
if ($mp->opts->tcp) { foreach (split(/\,/x, $mp->opts->tcp)) { $ports{"$_/tcp"} = 1 } }
if ($mp->opts->udp) { foreach (split(/\,/x, $mp->opts->udp)) { $ports{"$_/udp"} = 1 } }

# Delete open ports from data structure
foreach (@{$nmapxml->{'host'}->{'ports'}->{'port'}}) {
  if ($_->{'state'}->{'state'} eq 'open') {
    delete($ports{$_->{'portid'}.'/'.$_->{'protocol'}})
  }
}

# Check if any ports are left in the hash
if (keys(%ports) > 0) {
  $mp->plugin_exit(CRITICAL, "These ports are not open: ".join(", ", keys(%ports)));
} else {
  $mp->plugin_exit(OK, "All ports are open")
}

# This point should not be reached
exit(4);

sub port_range {
  my ($tcp, $udp) = @_;
  my $tcpstring = '';
  my $udpstring = '';
  
  if ($tcp) { $tcpstring .= "T:$tcp" }
  if ($udp) { $udpstring .= "U:$udp" }

  if ( $tcp && (not($udp)) ) { return $tcpstring }
  if ( $udp && (not($tcp)) ) { return $udpstring }
  return $tcpstring.",".$udpstring;
}

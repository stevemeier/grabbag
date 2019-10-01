#!/usr/bin/perl

use strict;
use warnings;
use Data::Dumper;
use Frontier::Client;

my $client = new Frontier::Client(url => "http://localhost/rpc/api");
my $session = $client->call('auth.login', 'admin', 'admin1');

my $actions = $client->call('schedule.list_archived_actions', $session);

foreach my $action (@$actions) {
  print '"'.$action->{name}."\",";
  printf("%d-%02d-%02d%s%s", unpack('A4A2A2AA8', $action->{earliest}->value() ));
  print ',';
  
  my $scriptdetails = $client->call('system.get_script_action_details', $session, $action->{id});
  print '"'.$scriptdetails->{content}.'"';
  print "\n";
}

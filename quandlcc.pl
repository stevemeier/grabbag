#!/usr/bin/perl
#
# Author:   Steve Meier
# Homepage: http://www.steve-meier.de
# Date:     2017-11-04
#
# Currency converter script based on Quandl.com API
# 
# If you want to make a bunch of request, get yourself an API key

use strict;
use warnings;
use Getopt::Long;
use JSON;
use LWP::UserAgent;

# Put your key in, if you have one
my $apikey = '';
my $baseurl = "https://www.quandl.com/api/v3/datasets/";

# Parse command line options
my ($value, $in, $out, $debug, $test);
GetOptions( "in=s"  => \$in,
	    "out=s" => \$out,
            "debug" => \$debug,
            "test"  => \$test );

# Ensure that currency codes are lower-case
$in  = lc($in);
$out = lc($out);

# The remaining parameter should be the amount
if (not(defined($ARGV[0]))) {
  print "No value to convert\n";
  exit 1;
} else {
  $value = $ARGV[0];
}

# Setup LWP objects
my $ua = LWP::UserAgent->new(timeout => 10, agent => 'curl/7.43.0');
   $ua->default_header('Accept' => 'application/json');
my $uadata;

# Set up JSON objects
my $json = JSON->new->allow_nonref;
my $jsondata;

# Map of API
my %apimap = ( 'gbp' => { 'aud' => 'BOE/XUDLADS',
	                  'cad' => 'BOE/XUDLCDS',
	                  'cny' => 'BOE/XUDLBK89',
	                  'czk' => 'BOE/XUDLBK25',
	                  'dkk' => 'BOE/XUDLDKS',
	                  'hkd' => 'BOE/XUDLHDS',
	                  'huf' => 'BOE/XUDLBK33',
	                  'inr' => 'BOE/XUDLBK97',
	                  'nis' => 'BOE/XUDLBK78',
	                  'jpy' => 'BOE/XUDLJYS',
	                  'myr' => 'BOE/XUDLBK83',
	                  'nzd' => 'BOE/XUDLNDS',
	                  'nok' => 'BOE/XUDLNKS',
	                  'pln' => 'BOE/XUDLBK47',
	                  'rub' => 'BOE/XUDLBK85',
	                  'sar' => 'BOE/XUDLSRS',
	                  'sgd' => 'BOE/XUDLSGS',
	                  'zar' => 'BOE/XUDLZRS',
	                  'krw' => 'BOE/XUDLBK93',
	                  'sek' => 'BOE/XUDLSKS',
	                  'chf' => 'BOE/XUDLSFS',
	                  'twd' => 'BOE/XUDLTWS',
	                  'thb' => 'BOE/XUDLBK87',
	                  'try' => 'BOE/XUDLBK95' },

	       'usd' => { 'brl' => 'FRED/DEXBZUS',
		          'cad' => 'FRED/DEXCAUS',
		          'cny' => 'FRED/DEXCHUS',
		          'dkk' => 'FRED/DEXDNUS',
		          'hkd' => 'FRED/DEXHKUS',
		          'inr' => 'FRED/DEXINUS',
		          'jpy' => 'FRED/DEXJPUS',
	                  'myr' => 'FRED/DEXMAUS',
	                  'mxn' => 'FRED/DEXMXUS',
	                  'twd' => 'FRED/DEXTAUS',
	                  'nok' => 'FRED/DEXNOUS',
	                  'sgd' => 'FRED/DEXSIUS',
	                  'zar' => 'FRED/DEXSFUS',
	                  'krw' => 'FRED/DEXKOUS',
	                  'lkr' => 'FRED/DEXSLUS',
	                  'sek' => 'FRED/DEXSDUS',
	                  'chf' => 'FRED/DEXSZUS',
	                  'thb' => 'FRED/DEXTHUS',
	                  'vef' => 'FRED/DEXVZUS' },
	       'aud' => { 'usd' => 'FRED/DEXUSAL' },
	       'nzd' => { 'usd' => 'FRED/DEXUSNZ' },

	       'eur' => { 'aud' => 'ECB/EURAUD',
                          'bgn' => 'ECB/EURBGN',
          	          'brl' => 'ECB/EURBRL',
	                  'cad' => 'ECB/EURCAD',
	                  'chf' => 'ECB/EURCHF',
	                  'cny' => 'ECB/EURCNY',
	                  'czk' => 'ECB/EURCZK',
	                  'dkk' => 'ECB/EURDKK',
	                  'gbp' => 'ECB/EURGBP',
	                  'hkd' => 'ECB/EURHKD',
	                  'hrk' => 'ECB/EURHRK',
	                  'huf' => 'ECB/EURHUF',
	                  'idr' => 'ECB/EURIDR',
	                  'ils' => 'ECB/EURILS',
	                  'inr' => 'ECB/EURINR',
	                  'isk' => 'ECB/EURISK',
	                  'jpy' => 'ECB/EURJPY',
	                  'krw' => 'ECB/EURKRW',
	                  'mxn' => 'ECB/EURMXN',
	                  'myr' => 'ECB/EURMYR',
	                  'nok' => 'ECB/EURNOK',
	                  'nzd' => 'ECB/EURNZD',
	                  'php' => 'ECB/EURPHP',
	                  'pln' => 'ECB/EURPLN',
	                  'ron' => 'ECB/EURRON',
	                  'rub' => 'ECB/EURRUB',
	                  'sek' => 'ECB/EURSEK',
	                  'sgd' => 'ECB/EURSGD',
	                  'thb' => 'ECB/EURTHB',
	                  'try' => 'ECB/EURTRY',
	                  'usd' => 'ECB/EURUSD',
	                  'zar' => 'ECB/EURZAR' }
	     );

# Run through all conversions (for testing)
if ($test) {
  foreach my $l1 (sort(keys(%apimap))) {
    foreach my $l2 (sort(keys(%{$apimap{$l1}}))) {
      print "$l1 -> $l2\t";
      print &convert($value, $l1, $l2);
      sleep 1;
    }
  }
  exit;
}

# Do the single conversion
print &convert($value, $in, $out);
exit;

sub convert {
  my ($lvalue, $lin, $lout) = @_;
  my $multiplier;

  if (defined($apimap{$lin}{$lout})) {
    # We can do a straight conversion
    $multiplier = &get_latest($apimap{$lin}{$lout});
    $lvalue = $lvalue * $multiplier;
    return sprintf("%.2f\n", $lvalue);

  } else {
    # Check if we can do an inverted conversion
    foreach my $base (keys(%apimap)) {
      if ( (defined($apimap{$base}{$lin})) && ($base eq $lout) ) {

        # For the inverted conversion we do a division
        &debug("Doing inverted conversion\n");
        $multiplier = &get_latest($apimap{$base}{$lin});
        $lvalue = $lvalue / $multiplier;
        return sprintf("%.2f\n", $lvalue);
      }
    }
  }

  return;
}

sub get_latest {
  my $endpoint = shift;
  my $latest;

  if ($apikey) { 
    &debug("Using API key\n");
    $endpoint .= '?api_key='.$apikey }

  &debug("Calling $baseurl$endpoint\n");
  $uadata = $ua->get($baseurl.$endpoint);

  # Check if the request succeeded
  if ($uadata->is_success) {
    # Decode the JSON data
    $jsondata = $json->decode($uadata->content);

    # Find the latest element in the dataset
    $latest = $jsondata->{'dataset'}->{'newest_available_date'};
    foreach my $data (@{$jsondata->{'dataset'}->{'data'}}) {
      if (@{$data}[0] eq $latest) { 
	&debug("Latest data is from $latest\n");
	return @{$data}[1];
      }
    }
  } else {
    print STDERR "ERROR: Could not fetch $endpoint\n";
    exit 1;
  }

  return;
}

sub debug {
  if ($debug) { print STDERR "DEBUG: @_"; }

  return;
}

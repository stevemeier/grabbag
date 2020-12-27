#!/usr/bin/perl
#
# Author:   Steve Meier
# Homepage: http://www.steve-meier.de
# Date:     2017-11-04
#
# Currency converter script based on Quandl.com API
#
# If you want to make a bunch of requests, get yourself an API key

use strict;
use warnings;
use Getopt::Long;
use JSON;
use LWP::UserAgent;

# Put your key in, if you have one
my $apikey = '';
my $baseurl = "https://www.quandl.com/api/v3/datasets/";

# Parse command line options
my $in = "";
my $out = "";
my ($value, $debug, $test);
my $decimal = 2;
GetOptions( "in=s"      => \$in,
	    "out=s"     => \$out,
	    "decimal=s" => \$decimal,
            "debug"     => \$debug,
            "test"      => \$test );

# The remaining parameter should be the amount
if (not(defined($ARGV[0]))) {
  print "ERROR: No value to convert\n";
  &usage;
  exit 1;
} else {
  $value = $ARGV[0];
}

# Ensure that currency codes are lower-case
$in  = lc($in);
$out = lc($out);

# Setup LWP objects
my $ua = LWP::UserAgent->new(timeout => 10, agent => 'curl/7.43.0');
   $ua->default_header('Accept' => 'application/json');
my $uadata;

# Set up JSON objects
my $json = JSON->new->allow_nonref;
my $jsondata;

# Map of API
my %apimap = ( 'gbp' => { 'aud' => [ 'BOE/XUDLADS', 1 ],
	                  'cad' => [ 'BOE/XUDLCDS', 1 ],
	                  'cny' => [ 'BOE/XUDLBK89', 1 ],
	                  'czk' => [ 'BOE/XUDLBK25', 1 ],
	                  'dkk' => [ 'BOE/XUDLDKS', 1 ],
	                  'hkd' => [ 'BOE/XUDLHDS', 1 ],
	                  'huf' => [ 'BOE/XUDLBK33', 1 ],
	                  'inr' => [ 'BOE/XUDLBK97', 1 ],
	                  'nis' => [ 'BOE/XUDLBK78', 1 ],
	                  'jpy' => [ 'BOE/XUDLJYS', 1 ],
	                  'myr' => [ 'BOE/XUDLBK83', 1 ],
	                  'nzd' => [ 'BOE/XUDLNDS', 1 ],
	                  'nok' => [ 'BOE/XUDLNKS', 1 ],
	                  'pln' => [ 'BOE/XUDLBK47', 1 ],
	                  'rub' => [ 'BOE/XUDLBK85', 1 ],
	                  'sar' => [ 'BOE/XUDLSRS', 1 ],
	                  'sgd' => [ 'BOE/XUDLSGS', 1 ],
	                  'zar' => [ 'BOE/XUDLZRS', 1 ],
	                  'krw' => [ 'BOE/XUDLBK93', 1 ],
	                  'sek' => [ 'BOE/XUDLSKS', 1 ],
	                  'chf' => [ 'BOE/XUDLSFS', 1 ],
	                  'twd' => [ 'BOE/XUDLTWS', 1 ],
	                  'thb' => [ 'BOE/XUDLBK87', 1 ],
	                  'try' => [ 'BOE/XUDLBK95', 1 ] },

	       'usd' => { 'brl' => [ 'FRED/DEXBZUS', 1 ],
		          'cad' => [ 'FRED/DEXCAUS', 1 ],
		          'cny' => [ 'FRED/DEXCHUS', 1 ],
		          'dkk' => [ 'FRED/DEXDNUS', 1 ],
		          'hkd' => [ 'FRED/DEXHKUS', 1 ],
		          'inr' => [ 'FRED/DEXINUS', 1 ],
		          'jpy' => [ 'FRED/DEXJPUS', 1 ],
	                  'myr' => [ 'FRED/DEXMAUS', 1 ],
	                  'mxn' => [ 'FRED/DEXMXUS', 1 ],
	                  'twd' => [ 'FRED/DEXTAUS', 1 ],
	                  'nok' => [ 'FRED/DEXNOUS', 1 ],
	                  'sgd' => [ 'FRED/DEXSIUS', 1 ],
	                  'zar' => [ 'FRED/DEXSFUS', 1 ],
	                  'krw' => [ 'FRED/DEXKOUS', 1 ],
	                  'lkr' => [ 'FRED/DEXSLUS', 1 ],
	                  'sek' => [ 'FRED/DEXSDUS', 1 ],
	                  'chf' => [ 'FRED/DEXSZUS', 1 ],
	                  'thb' => [ 'FRED/DEXTHUS', 1 ],
	                  'vef' => [ 'FRED/DEXVZUS', 1 ] },
	       'aud' => { 'usd' => [ 'FRED/DEXUSAL', 1 ] },
	       'nzd' => { 'usd' => [ 'FRED/DEXUSNZ', 1 ] },

	       'eur' => { 'aud' => [ 'ECB/EURAUD', 1 ],
                          'bgn' => [ 'ECB/EURBGN', 1 ],
                          'brl' => [ 'ECB/EURBRL', 1 ],
	                  'cad' => [ 'ECB/EURCAD', 1 ],
	                  'chf' => [ 'ECB/EURCHF', 1 ],
	                  'cny' => [ 'ECB/EURCNY', 1 ],
	                  'czk' => [ 'ECB/EURCZK', 1 ],
	                  'dkk' => [ 'ECB/EURDKK', 1 ],
	                  'gbp' => [ 'ECB/EURGBP', 1 ],
	                  'hkd' => [ 'ECB/EURHKD', 1 ],
	                  'hrk' => [ 'ECB/EURHRK', 1 ],
	                  'huf' => [ 'ECB/EURHUF', 1 ],
	                  'idr' => [ 'ECB/EURIDR', 1 ],
	                  'ils' => [ 'ECB/EURILS', 1 ],
	                  'inr' => [ 'ECB/EURINR', 1 ],
	                  'isk' => [ 'ECB/EURISK', 1 ],
	                  'jpy' => [ 'ECB/EURJPY', 1 ],
	                  'krw' => [ 'ECB/EURKRW', 1 ],
	                  'mxn' => [ 'ECB/EURMXN', 1 ],
	                  'myr' => [ 'ECB/EURMYR', 1 ],
	                  'nok' => [ 'ECB/EURNOK', 1 ],
	                  'nzd' => [ 'ECB/EURNZD', 1 ],
	                  'php' => [ 'ECB/EURPHP', 1 ],
	                  'pln' => [ 'ECB/EURPLN', 1 ],
	                  'ron' => [ 'ECB/EURRON', 1 ],
	                  'rub' => [ 'ECB/EURRUB', 1 ],
	                  'sek' => [ 'ECB/EURSEK', 1 ],
	                  'sgd' => [ 'ECB/EURSGD', 1 ],
	                  'thb' => [ 'ECB/EURTHB', 1 ],
	                  'try' => [ 'ECB/EURTRY', 1 ],
	                  'usd' => [ 'ECB/EURUSD', 1 ],
	                  'zar' => [ 'ECB/EURZAR', 1 ] },

	       'xag' => { 'eur' => [ 'LBMA/SILVER', 3 ],
	                  'gbp' => [ 'LBMA/SILVER', 2 ],
	                  'usd' => [ 'LBMA/SILVER', 1 ] },

	       'xau' => { 'eur' => [ 'LBMA/GOLD', 6 ],
	                  'gbp' => [ 'LBMA/GOLD', 4 ],
	                  'usd' => [ 'LBMA/GOLD', 2 ] }
	     );

# Run through all conversions (for testing)
if ($test) {
  foreach my $l1 (sort(keys(%apimap))) {
    foreach my $l2 (sort(keys(%{$apimap{$l1}}))) {
      print "$l1 -> $l2\t";
      printf('%.'.$decimal."f\n", &convert($value, $l1, $l2));
      sleep 1;
    }
  }
  exit;
}

# Do the single conversion
printf('%.'.$decimal."f\n", &convert($value, $in, $out));
exit;

sub convert {
  my ($lvalue, $lin, $lout) = @_;
  my $multiplier;

  if (defined($apimap{$lin}{$lout})) {
    # We can do a straight conversion
    $multiplier = &get_latest(@{$apimap{$lin}{$lout}});
    $lvalue = $lvalue * $multiplier;
    return $lvalue;

  } else {
    # Check if we can do an inverted conversion
    foreach my $base (keys(%apimap)) {
      if ( (defined($apimap{$base}{$lin})) && ($base eq $lout) ) {

        # For the inverted conversion we do a division
        &debug("Doing inverted conversion\n");
        $multiplier = &get_latest(@{$apimap{$base}{$lin}});
        $lvalue = $lvalue / $multiplier;
        return $lvalue;
      }
    }
  }

  return;
}

sub get_latest {
  my @endpoint = @_;
  my $latest;
  my $result;

  &debug("Endpoint 0 -> $endpoint[0]\n");
  &debug("Endpoint 1 -> $endpoint[1]\n");

  if ($apikey) {
    &debug("Using API key $apikey\n");
    $endpoint[0] .= '?api_key='.$apikey
  }

  &debug("Calling $baseurl$endpoint[0]\n");
  $uadata = $ua->get($baseurl.$endpoint[0]);

  # Check if the request succeeded
  if ($uadata->is_success) {
    # Decode the JSON data
    $jsondata = $json->decode($uadata->content);

    # Find the latest element in the dataset
    $latest = $jsondata->{'dataset'}->{'newest_available_date'};
    foreach my $data (@{$jsondata->{'dataset'}->{'data'}}) {
      if (@{$data}[0] eq $latest) {
	&debug("Latest data is from $latest\n");
        $result = @{$data}[$endpoint[1]];

        # On Christmas and New Year's Eve PM is `null`, so return `AM` instead
        if ( ($endpoint[0] =~ /GOLD/) && (not(defined(@{$data}[$endpoint[1]]))) ) {
          $result = @{$data}[$endpoint[1]-1];
        }

	&debug("Returning $result\n");
	return $result;
      }
    }
  } else {
    print STDERR "ERROR: Could not fetch $endpoint[0]\n";
    exit 1;
  }

  return;
}

sub debug {
  if ($debug) { print STDERR "DEBUG: @_"; }

  return;
}

sub usage {
  print "Usage: $0 --in <CURRENCY> --out <CURRENCY> [ --decimal <n> ] [ --debug ] [ --test ] <VALUE>\n\n";
  print "CURRENCY is a ISO-4217 code such as EUR, USD, AUD, etc.\n";
  print "VALUE is the amount in the IN currency, which will be converted to OUT currency\n\n";
  print "--decimcal <n>\t\tProvide <n> decimal points (0 = integer)\n";
  print "--debug\t\t\tEnables debug output\n";
  print "--test\t\t\tTest all available conversions\n\n";
}

#!/usr/bin/perl

# Quick PoC to store and receives files via DNS
# Date: December 29th, 2017
# Author: Steve Meier
#
# History
# 20200228 - Added compression and --debug

# Load all required modules
use strict;
use warnings;
use Carp;
use Compress::Zlib;
use Digest::SHA;
use File::Basename qw(basename);
use Getopt::Long;
use MIME::Base64 qw(decode_base64 encode_base64);
use Net::DNS;

my ($encode, $download, $dnsname, $output, $ttl, $hashalg, $debug);
GetOptions("encode=s"   => \$encode,
	   "download"   => \$download,
           "dnsname=s"  => \$dnsname,
           "output=s"   => \$output,
           "ttl=s"      => \$ttl,
           "hashalg=s"  => \$hashalg,
           "debug"      => \$debug);

if (not(defined($hashalg))) { $hashalg = "sha256" };
	 
# Encode a file into Base64 and DNS TXT records   
if ($encode) {
  my $base64data;
  my $txtdata;
  my $i = 0;
  my $hash;
  my $size;
  my $name;

  # Default TTL. Can be overridden with --ttl
  if (not(defined($ttl))) { $ttl = 86400 };

  if (not(-f $encode)) {
    print STDERR "ERROR: Please specify a file to encode!\n";
    exit 1;
  }

  if (not(defined($dnsname))) {
    print STDERR "ERROR: Please specify DNS name!\n";
    exit 2;
  }

  # Ensure we use a fully qualified name, ending with dot
  if (not($dnsname =~ /\.$/x)) { $dnsname .= "." };

  # We store meta information such as original file name, size and checksum
  $name = basename($encode);
  $size = filesize($encode);
  $hash = filehash($encode, $hashalg);

  # Read the file in one go
  open(my $FILE, "<", $encode) || croak "ERROR: Could not read file!\n";
  local($/) = undef;
  $base64data = encode_base64(compress(<$FILE>), '');
  close($FILE) || croak "ERROR: Could not close file!\n";

  # Print the base record with meta information
  print "$dnsname $ttl IN TXT \"$name\" \"$size\" \"$hashalg\" \"$hash\"\n";

  # Print the data records
  while (length($base64data) > 0) {
    $txtdata = substr $base64data, 0, 255, '';
    print i_to_label($i).".$dnsname $ttl IN TXT \"$txtdata\"\n";
    $i++;
  }

  exit 0;
}

if ($download) {
  my ($reply, $rr);
  my @filemeta;
  my $base64data;
  my $i = 0;
  my $filename;

  # Create a resolver object
  my $res = Net::DNS::Resolver->new(adflag => 1, debug => $debug);

  # Look up the files meta information (name, size, hashalg, hash)
  $reply = $res->query($dnsname, 'TXT');
  if ($reply) {
    foreach my $rr (grep { $_->type eq 'TXT' } $reply->answer) {
      @filemeta = $rr->txtdata;
    }
  }

  if ($#filemeta < 0) {
    print STDERR "ERROR: No metadata found in DNS\n";
    exit 1;
  }
  &debug(join("\t", @filemeta)."\n");

  # Save to original or provided filename
  if ($output) {
    $filename = $output;
  } else {
    $filename = $filemeta[0];
  }

  # Check if file is already up-to-date
  if (-f $filename) {
    if (filehash($filename, $filemeta[2]) eq $filemeta[3]) {
      print STDERR "INFO: File is already up-to-date\n";
      exit 0;
    }
  }

  # Do lookups until we encounter "nxdomain" which indrectly marks end of file
  until ($res->errorstring =~ /nxdomain/i) {
    $reply = $res->query(i_to_label($i).".$dnsname", 'TXT');
    if ($reply) {
      foreach my $rr (grep { $_->type eq 'TXT' } $reply->answer) {
        $base64data .= $rr->txtdata;
      }
    }
    $i++;
  }

  # Write base64 decoded file to disk
  open(my $FILE, ">", $filename) || croak "ERROR: Could not open $filename for writing";
  print $FILE uncompress(decode_base64($base64data));
  close($FILE) || croak "ERROR: Could not close $filename";
  
  # Check file size is correct
  if (filesize($filename) != $filemeta[1]) {
    print STDERR "ERROR: File size does not match!\n";
    exit 1;
  } 

  # Check file hash is correct
  if (filehash($filename, $filemeta[2]) ne $filemeta[3]) {
    print STDERR "ERROR: File hash does not match!\n";
    exit 2;
  }

  print STDERR "INFO: File downloaded successfully to $filename\n";
  exit 0;
}

print "Encode a file to zone file format:\n";
print "$0 --encode <FILE> --dnsname <somename.foobar.com> [ --hashalg <sha...> --debug ]\n\n";
print "Download a file from DNS:\n";
print "$0 --download --dnsname <somename.foobar.com> [ --output <FILE> --debug ]\n\n";
print "Hint: To use a specific nameserver, set the environment variable RES_NAMESERVERS\n";
exit;

# based on https://stackoverflow.com/a/12823896/1592267
sub i_to_label {
  my ($val) = @_;
  my $symbols = join('', 'a'..'z');
  my $b26 = '';
  while ($val) {
    $b26 = substr($symbols, $val % 26, 1) . $b26;
    $val = int $val / 26;
  }
  return $b26 || 'a';
}

sub filehash {
  my ($filename, $hashalg) = @_;

  my $sha = Digest::SHA->new($hashalg);
  $sha->addfile($filename);

  return $sha->hexdigest;
}

sub filesize {
  my ($filename) = @_;

  my $size = (stat $filename)[7];

  return $size;
}

sub debug {
  my ($message) = shift;

  if ($debug) {
    print "DEBUG: $message";
  }
}


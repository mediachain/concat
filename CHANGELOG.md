## 13/2/2017: concat-v1.6
- Directory extensions for manifest lookup [[PR #125]](https://github.com/mediachain/concat/pull/125)
- Peer discovery through DHT rendezvous [[PR #128]](https://github.com/mediachain/concat/pull/128)
- mcid usabililty improvements [[PR #124]](https://github.com/mediachain/concat/pull/124)
- Statement DB vacuum support [[PR #130]](https://github.com/mediachain/concat/pull/130)
- Misc fixes and improvements
  - version command for binaries [[PR #122]](https://github.com/mediachain/concat/pull/122)
  - Add LIMIT clause to MCQL DELETE [[PR #129]](https://github.com/mediachain/concat/pull/129)
  - Update rocksdb, disable concurrent memtable writing which is no longer supported with point lookup optimizations [[PR #126]](https://github.com/mediachain/concat/pull/126)

## 17/1/2017: concat-v1.5
- Mediachain identity [[PR #110]](https://github.com/mediachain/concat/pull/110) [[PR #112]](https://github.com/mediachain/concat/pull/112) [[PR #118]](https://github.com/mediachain/concat/pull/118)
- Node manifests [[PR #116]](https://github.com/mediachain/concat/pull/116)
- Improved IPFS interopability [[PR #113]](https://github.com/mediachain/concat/pull/113)
- Misc fixes and improvements
  - Deduplicate wkis in compound statements [[PR #107]](https://github.com/mediachain/concat/pull/107) [[PR #108]](https://github.com/mediachain/concat/pull/108)
  - Parallelize directory and DHT lookups, with more reasonable timeouts [[PR #109]](https://github.com/mediachain/concat/pull/109)
  - Canonical JSON marshalling for statements in query results [[PR #111]](https://github.com/mediachain/concat/pull/111)

## 19/12/2016: concat-v1.4
- Directory extensions for namespace listing [[PR #101]](https://github.com/mediachain/concat/pull/101)
- Multiple directories [[PR #102]](https://github.com/mediachain/concat/pull/102)
- Batch object retrieval api [[PR #100]](https://github.com/mediachain/concat/pull/100)
- Misc fixes and debugging
  - Fix build issues with gorocksdb [[PR #96]](https://github.com/mediachain/concat/pull/96)
  - Net connection api, consistent handle formatting [[PR #97]](https://github.com/mediachain/concat/pull/97)
  - Improved logging [[PR #103]](https://github.com/mediachain/concat/pull/103)
  - Fix ticker lick in go-libp2p-kad-dht [[PR #104]](https://github.com/mediachain/concat/pull/104)
  - Tune directory lookup timeouts for timely dht lookup fallback on directory failures [[PR #105]](https://github.com/mediachain/concat/pull/105)

## 6/12/2016: concat-v1.3.1
- Bug fix: crash when setting auth rules [[Issue #93]](https://github.com/mediachain/concat/issues/93)

## 5/12/2016: concat-v1.3
- Datastore garbage collection [[PR #88]](https://github.com/mediachain/concat/pull/88)
- IPFS DHT integration [[PR #90]](https://github.com/mediachain/concat/pull/90)

## 21/11/2016: concat-v1.2
- Push publishing for authorized peers [[PR #82]](https://github.com/mediachain/concat/pull/82)
- More robust public IP address detection [[PR #84]](https://github.com/mediachain/concat/pull/84)
- Multiaddr compatible peer handles [[PR #79]](https://github.com/mediachain/concat/pull/79)
- Improved peer lookup logic with libp2p-reported connectedness [[PR #75]](https://github.com/mediachain/concat/pull/75)

## 4/11/2016: concat-v1.1
- New directory: `/ip4/52.7.126.237/tcp/9000/QmSdJVceFki4rDbcSrW7JTJZgU9so25Ko7oKHE97mGmkU6`
- Add a deps field to statement body in order to support object dependency merging [[Issue #63]](https://github.com/mediachain/concat/issues/63)
- Optimize data fetching in the merge protocol [[Issue #44]](https://github.com/mediachain/concat/issues/44)
- Support compound statement publication [[Issue #62]](https://github.com/mediachain/concat/issues/62)
- Implement user write locking in statement db, allow concurrent reads with long running queries [[PR #73]](https://github.com/mediachain/concat/pull/73)
- Add delay in the offline->public transition when NAT config is auto, allow port mapping to complete before directory registration [[PR #72]](https://github.com/mediachain/concat/pull/72)
- Automate binary builds for releases [[Issue 56]](https://github.com/mediachain/concat/issues/56)

## 24/10/2016: concat-v1.0
- First release; baseline feature completeness.

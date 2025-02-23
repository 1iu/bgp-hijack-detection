package main

import (
	"context"
	"fmt"
	"github.com/ammario/ipisp/v2"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const originOfLessSpecific = 0
const sameSubnet = 1
const originOfMoreSpecific = 2

var recentFilesCounter = 1

var originCounters map[uint32]*originCounter

type originCounter struct {
	asn                 uint32
	counterLessSpecific uint32 //potential victim
	counterMoreSpecific uint32 //potential attacker
	counterSameSubnet   uint32 //

	counterLessSpecificRecent    uint32 //potential victim
	counterMoreSpecificRecent    uint32 //potential attacker
	counterSameSubnetRecent      uint32 //
	topographicallyRelatedRecent uint32

	country                string
	registry               string
	isp                    string
	topographicallyRelated uint32
}

func updateOriginCounter(asn uint32, level uint32, inPathofOther bool) {
	mutex.Lock()
	v, ok := originCounters[asn]
	if ok {
		switch level {
		case originOfLessSpecific:
			v.counterLessSpecific++
			v.counterLessSpecificRecent++
		case sameSubnet:
			v.counterSameSubnet++
			v.counterSameSubnetRecent++
		case originOfMoreSpecific:
			v.counterMoreSpecific++
			v.counterMoreSpecificRecent++
		}
		if inPathofOther {
			v.topographicallyRelated++
			v.topographicallyRelatedRecent++
		}

	} else {
		country := "-"
		registry := "-"
		isp := "-"
		resp, err := ipisp.LookupASN(context.Background(), ipisp.ASN(asn))
		if err != nil {
			fmt.Println(Red("Lookup for ASN: ", asn, " did not work: ", err))
		} else {
			country = resp.Country
			registry = resp.Registry
			isp = strings.ReplaceAll(resp.ISPName, ",", "")
		}

		newCounter := originCounter{
			asn:      asn,
			country:  country,
			registry: registry,
			isp:      isp,
		}
		switch level {
		case originOfLessSpecific:
			newCounter.counterLessSpecific++
			newCounter.counterLessSpecificRecent++
		case sameSubnet:
			newCounter.counterSameSubnet++
			newCounter.counterSameSubnetRecent++
		case originOfMoreSpecific:
			newCounter.counterMoreSpecific++
			newCounter.counterMoreSpecificRecent++
		}
		if inPathofOther {
			newCounter.topographicallyRelated++
			newCounter.topographicallyRelatedRecent++
		}
		originCounters[asn] = &newCounter

	}
	mutex.Unlock()
}

func checkForTimeIntervall(intervall int) {
	for range time.Tick(time.Minute * time.Duration(intervall)) {
		fmt.Println(Teal("Printing next rough overview of origin AS activity in the last ", strconv.Itoa(intervall), " minutes. [", time.Now().String(), "]"))
		writeRecentOriginFrequencies()
		mutex.Lock()
		for as := range originCounters {
			originCounters[as].counterSameSubnetRecent = 0
			originCounters[as].counterMoreSpecificRecent = 0
			originCounters[as].counterLessSpecificRecent = 0
			originCounters[as].topographicallyRelatedRecent = 0
		}
		mutex.Unlock()
		fmt.Println(Teal("Finished printing of rough overview of most recent origin ASes activities. [", time.Now().String(), "]"))

	}
}

func updateSummary(c conflicts) {
	m1 := c.referenceAnnouncement
	for _, m2 := range c.conflictingMessages {
		m1OriginInM2path := IsContainedInUint32(m2.aspath, m1.origin)
		m2OriginInM1path := IsContainedInUint32(m1.aspath, m2.origin)

		oneOriginInASpathOfOther := m1OriginInM2path || m2OriginInM1path

		if len(m1.subnetAsBits) == len(m2.subnetAsBits) {
			updateOriginCounter(m1.origin, sameSubnet, oneOriginInASpathOfOther)
			updateOriginCounter(m2.origin, sameSubnet, oneOriginInASpathOfOther)
		}
		if len(m1.subnetAsBits) > len(m2.subnetAsBits) {
			updateOriginCounter(m1.origin, originOfMoreSpecific, oneOriginInASpathOfOther)
			updateOriginCounter(m2.origin, originOfLessSpecific, oneOriginInASpathOfOther)
		}
		if len(m1.subnetAsBits) < len(m2.subnetAsBits) {
			updateOriginCounter(m1.origin, originOfLessSpecific, oneOriginInASpathOfOther)
			updateOriginCounter(m2.origin, originOfMoreSpecific, oneOriginInASpathOfOther)
		}

	}
}

func printShortSummary() {
	asSlice := make([]uint32, 0, len(originCounters))
	for as := range originCounters {
		asSlice = append(asSlice, as)
	}

	fmt.Println()

	fmt.Println(Teal("----------------------------------------------------------------------------------------------------------------------------------"))
	fmt.Println(Teal("Short Summary of mostly involved ASes"))
	fmt.Println()
	fmt.Println(White("Number of ASes involved in conflicts: ", len(originCounters)))
	fmt.Println()
	fmt.Println(Teal("Most often as origin in the less specific message (potential victim)"))
	sort.Slice(asSlice, func(i, j int) bool {
		return originCounters[asSlice[i]].counterLessSpecific > originCounters[asSlice[j]].counterLessSpecific
	})
	printTopAS(10, asSlice)

	fmt.Println()
	fmt.Println(Teal("Most often as origin in the more specific message (potential attacker)"))
	sort.Slice(asSlice, func(i, j int) bool {
		return originCounters[asSlice[i]].counterMoreSpecific > originCounters[asSlice[j]].counterMoreSpecific
	})
	printTopAS(10, asSlice)

	fmt.Println()
	fmt.Println(Teal("Most often as origin in conflict with conflicting announcements for same subnet"))
	sort.Slice(asSlice, func(i, j int) bool {
		return originCounters[asSlice[i]].counterSameSubnet > originCounters[asSlice[j]].counterSameSubnet
	})
	printTopAS(10, asSlice)

	fmt.Println()
	fmt.Println(Teal("Most often as origin in conflicts without topological relation based on AS path"))
	sort.Slice(asSlice, func(i, j int) bool {
		return originCounters[asSlice[i]].counterMoreSpecific+originCounters[asSlice[i]].counterSameSubnet+originCounters[asSlice[i]].counterLessSpecific-originCounters[asSlice[i]].topographicallyRelated > originCounters[asSlice[j]].counterMoreSpecific+originCounters[asSlice[j]].counterSameSubnet+originCounters[asSlice[j]].counterLessSpecific-originCounters[asSlice[j]].topographicallyRelated
	})
	printTopAS(10, asSlice)

}

func printTopAS(n int, asSlice []uint32) {
	for i := 0; i < min(n, len(asSlice)); i++ {
		counterTotal := originCounters[asSlice[i]].counterLessSpecific + originCounters[asSlice[i]].counterSameSubnet + originCounters[asSlice[i]].counterMoreSpecific
		legitPercentage := (float32(originCounters[asSlice[i]].topographicallyRelated) / float32(counterTotal)) * 100
		fmt.Println(White("AS ", asSlice[i],
			" -> victim: ", originCounters[asSlice[i]].counterLessSpecific, ", same subnet: ", originCounters[asSlice[i]].counterSameSubnet, ", attacker: ", originCounters[asSlice[i]].counterMoreSpecific,
			" [total: ", counterTotal, ", legit ", legitPercentage, "%]"))

		resp, err := ipisp.LookupASN(context.Background(), ipisp.ASN(asSlice[i]))
		if err != nil {
			println(Red("Lookup for ASN: ", asSlice[i], " did not work: ", err))
			continue
		}
		fmt.Printf("   [%+v, %+v]:  %+v  \n\n", resp.AllocatedAt.Format("2006-02-01"), resp.Registry, resp.ISPName)
	}
}

func writeOriginFrequencies() {
	mutex.Lock()
	asSlice := make([]uint32, 0, len(originCounters))
	for as := range originCounters {
		asSlice = append(asSlice, as)
	}
	sort.Slice(asSlice, func(i, j int) bool {
		return originCounters[asSlice[i]].counterMoreSpecific+originCounters[asSlice[i]].counterSameSubnet+originCounters[asSlice[i]].counterLessSpecific > originCounters[asSlice[j]].counterMoreSpecific+originCounters[asSlice[j]].counterSameSubnet+originCounters[asSlice[j]].counterLessSpecific
	})

	var err error
	originsFile, err = os.Create(originsFileName + ".csv")
	if err != nil {
		fmt.Println(Red("could not create csv file for frequencies of origin ASes"))
		return
	}
	_, err = originsFile.WriteString("asn,total,lessSpecificOrigin,sameSubnet,moreSpecificOrigin,legit,registry,country,ispName\n")
	if err != nil {
		fmt.Println(Red("Could not write to originsFile"))
		return
	}

	for i := 0; i < len(asSlice); i++ {
		counterTotal := originCounters[asSlice[i]].counterLessSpecific + originCounters[asSlice[i]].counterSameSubnet + originCounters[asSlice[i]].counterMoreSpecific
		//legitPercentage := originCounters[asSlice[i]].topographicallyRelated / counterTotal
		_, err = originsFile.WriteString("" + strconv.Itoa(int(originCounters[asSlice[i]].asn)) + "," + strconv.Itoa(int(counterTotal)) + "," + strconv.Itoa(int(originCounters[asSlice[i]].counterLessSpecific)) + "," + strconv.Itoa(int(originCounters[asSlice[i]].counterSameSubnet)) + "," + strconv.Itoa(int(originCounters[asSlice[i]].counterMoreSpecific)) + "," + strconv.Itoa(int(originCounters[asSlice[i]].topographicallyRelated)) +
			"," + originCounters[asSlice[i]].registry + "," + originCounters[asSlice[i]].country + "," + originCounters[asSlice[i]].isp + "\n")
		if err != nil {
			fmt.Println(Red("Could not write to originsFile"))
			return
		}

	}
	defer originsFile.Close()
	mutex.Unlock()

}

func writeRecentOriginFrequencies() {
	mutex.Lock()
	asSlice := make([]uint32, 0, len(originCounters))
	for as := range originCounters {
		asSlice = append(asSlice, as)
	}
	sort.Slice(asSlice, func(i, j int) bool {
		return originCounters[asSlice[i]].counterMoreSpecificRecent+originCounters[asSlice[i]].counterSameSubnetRecent+originCounters[asSlice[i]].counterLessSpecificRecent > originCounters[asSlice[j]].counterMoreSpecificRecent+originCounters[asSlice[j]].counterSameSubnetRecent+originCounters[asSlice[j]].counterLessSpecificRecent
	})

	var err error
	originsFile, err = os.Create(originsFileName + strconv.Itoa(recentFilesCounter) + ".csv")
	recentFilesCounter++
	if err != nil {
		fmt.Println(Red("could not create csv file for frequencies of origin ASes"))
		return
	}
	_, err = originsFile.WriteString("asn,total,lessSpecificOrigin,sameSubnet,moreSpecificOrigin,legit,registry,country,ispName\n")
	if err != nil {
		fmt.Println(Red("Could not write to originsFile"))
		return
	}

	for i := 0; i < len(asSlice); i++ {
		counterTotal := originCounters[asSlice[i]].counterLessSpecificRecent + originCounters[asSlice[i]].counterSameSubnetRecent + originCounters[asSlice[i]].counterMoreSpecificRecent
		//legitPercentage := originCounters[asSlice[i]].topographicallyRelated / counterTotal
		_, err = originsFile.WriteString("" + strconv.Itoa(int(originCounters[asSlice[i]].asn)) + "," + strconv.Itoa(int(counterTotal)) + "," + strconv.Itoa(int(originCounters[asSlice[i]].counterLessSpecificRecent)) + "," + strconv.Itoa(int(originCounters[asSlice[i]].counterSameSubnetRecent)) + "," + strconv.Itoa(int(originCounters[asSlice[i]].counterMoreSpecificRecent)) + "," + strconv.Itoa(int(originCounters[asSlice[i]].topographicallyRelatedRecent)) +
			"," + originCounters[asSlice[i]].registry + "," + originCounters[asSlice[i]].country + "," + originCounters[asSlice[i]].isp + "\n")
		if err != nil {
			fmt.Println(Red("Could not write to originsFile"))
			return
		}

	}
	defer originsFile.Close()
	mutex.Unlock()
}

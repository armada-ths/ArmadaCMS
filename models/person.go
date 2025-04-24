package models

// Person represents an individual in the organization
type Person struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Email       *string `json:"email"`
    Picture     *string  `json:"picture"`
    LinkedInURL *string `json:"linkedin_url"`
    Role        string  `json:"role"`
}

// OrganizationGroup represents a single organization group with its members
type OrganizationGroup struct {
    Name   string   `json:"name"`
    People []Person `json:"people"`
}

// TODO: Remove these things after endpoint is implemented
//
//	mux.HandleFunc("/api/v1/organization", ).Methods("GET")

// GetOrganizationGroups returns a sample list of organization groups
func GetOrganizationGroups() []OrganizationGroup {
    ptr := func(s string) *string { return &s }
    // Create multiple organization groups
    return []OrganizationGroup{
        {
            Name: "Business Relations & Events",
            People: []Person{
                {
                    ID:          1,
                    Name:        "Paresh Sakala Vishwanath",
                    Email:       ptr("paresh.vishwanath@armada.nu"),
                    Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/4a8212359c2b44048afd3d4fb235b60f.jpg"),
                    LinkedInURL: ptr("https://www.linkedin.com/in/paresh-sakala-vishwanath-8519871a1"),
                    Role:        "Project Group–Head of Business Relations & Events",
                },
                {
                    ID:          2,
                    Name:        "Guantong Liu",
                    Email:       ptr("guantong.liu@armada.nu"),
                    Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/0490b2c285f841b0b48147c9f654eb00.JPG"),
                    LinkedInURL: ptr("https://www.linkedin.com/in/guantong-vicky-liu-078856213/"),
                    Role:        "Project Group–Head of Events",
                },
				{
					ID:          3,
					Name:        "Melina Taheri",
                    Email:       ptr("melina.taheri@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/284c952e9da04991aa847ea18647f689.jpeg"),
                    LinkedInURL: ptr("https://se.linkedin.com/in/melina-taheri-526a8b286"),
					Role:        "Project Group–Head of Sales",
				},
				{
					ID:          4,
					Name:        "Prasanth Premnath",
                    Email:       ptr("prasanth.premnath@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/f9cd454d7f604c75abdf048bf4f90a6c.jpg"),
                    LinkedInURL: ptr("https://www.linkedin.com/in/prasanth-premnath-6bba7818b/"),
					Role:        "Project Group–Head of Sales",
				},
				{
					ID:          5,
					Name:        "Sofia Tang",
					Email:       ptr("sofia.tang@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/f31bb7c0c4a2455b83b3e389fa367d88.png"),
					LinkedInURL: ptr("https://linkedin.com/in/sofiaatang"),
					Role:        "Project Group–Head of Sales",
				},
				{
					ID:          6,
					Name: 	  	 "Tanmay Pisal",
                    Email:       ptr("tanmay.pisal@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/430dd3fd38b9457cb3c35aadd2536a6d.jpg"),
					LinkedInURL: ptr("https://www.linkedin.com/in/tanmaypisal"),
					Role:        "Project Group–Head of Sales",
				},
            },
        },
        {
            Name: "Fair & Logistics",
            People: []Person{
                {
                    ID:          7,
                    Name:        "Aengus Donnelly",
                    Email:       nil,
                    Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/375a944f48ad421db31ec5b2bf20877e.jpg"),
                    LinkedInURL: ptr("https://www.linkedin.com/in/aengus-donnelly-127a0a2b2/"),
                    Role:        "Project Group–Head of Career Fair",
                },
				{
                    ID:          8,
                    Name:        "Jennika Bjurshagen",
                    Email:       ptr("jennika.bjurshagen@armada.nu"),
                    Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/958004539df84b89aebb01a817311fb9.JPG"),
                    LinkedInURL: ptr("http://www.linkedin.com/in/jennika-bjurshagen"),
                    Role:        "Project Group–Head of Career Fair",
                },
				{
                    ID:          9,
                    Name:        "Leo Kihlström Durvall",
                    Email:       nil,
                    Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/689c4a6af22449b3bdea978c81fd3132.png"),
                    LinkedInURL: ptr("https://www.linkedin.com/in/leo-kihlstr%C3%B6m-durvall-08453a294/"),
                    Role:        "Project Group–Head of Fair & Logistics",
                },
				{
					ID:          10,
					Name:        "Oskar Tyllström",
					Email:       nil,
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/2147d95151e143aebca9968f305ea7a1.jpeg"),				
					LinkedInURL: nil,
					Role:        "Project Group–Head of Logistics",
				},
				{
					ID:          11,
					Name:        "Erik Hyslop",
					Email:       nil,
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/7219a1f49c214e5fa62256c5702cf7a0.JPG"),
					LinkedInURL: ptr("https://www.linkedin.com/in/erik-hyslop-041066304/"),
					Role:        "Project Group–Head of Service & Sponsorship",
				},
				{
					ID:          12,
					Name:        "Kanagambujam Venkatramani",
					Email: 	 	 ptr("kanagambujam.venkatramani@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/d8ef339e42d149afb74bdd42a387d4ca.JPG"),
					LinkedInURL: ptr("https://www.linkedin.com/in/kanaga-kth/"),
					Role:        "Project Group– Head of Sustainability",
				},
            },
        },
		{
			Name: "Human Resources & Diversity",
			People: []Person{
				{
					ID:          13,
					Name:        "Rima Nima Odeh",
					Email:       ptr("rima.nimahodeh@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/17905ee551a64b48b6c251cdb9c2788f.jpg"),
					LinkedInURL: ptr("https://www.linkedin.com/in/rima-nimah-odeh-669226272/"),
					Role:        "Project Group–Head of Banquet",
				},
				{
					ID:          14,
					Name:        "Agastheeswar Bommaraj",
					Email:       ptr("agastheeswar.bommaraj@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/a0f74f9b94ad4cad9523e39be7d64004.png"),
					LinkedInURL: ptr("https://www.linkedin.com/in/agastheeswar/"),
					Role:        "Project Group–Head of Human Resources & Diversity",
				},
				{
					ID:          15,
					Name: 	  	 "Hari Prasad Srinivasan",
					Email:       nil,
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/145d8f904459424fa5eb84ac2a6261aa.png"),
					LinkedInURL: ptr("https://www.linkedin.com/in/hariprasads1011/"),
					Role: 	     "Project Group-Head of Interal Events",
				},
			},
		},
		{
			Name: "Marketing & Communications",
			People: []Person{
				{
					ID:          16,
					Name:        "Mounika Venkatasami Babu",
					Email:       nil,
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/e93cab9352a54009af31741317f079f0.jpg"),
					LinkedInURL: ptr("https://www.linkedin.com/in/mounika-venkatasami-285388217/"),
					Role:        "Project Group–Head of Creative",
				},
				{
					ID:          17,
					Name:        "Carl Chemnitz",
					Email:       ptr("carl.chemnitz@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/3aba4ba4692640bd813db4fb3527978d.jpg"),
					LinkedInURL: ptr("https://www.linkedin.com/in/cchemnitz/"),
					Role:        "Project Group–Head of Marketing & Communications",
				},
				{
					ID:          18,
					Name:        "Iris Wirström",
					Email:       ptr("iris.wirstrom@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/66a65b3bb54542b3b804952624175025.jpg"),
					LinkedInURL: ptr("https://www.linkedin.com/in/iris-wirstr%C3%B6m-363007235/"),
					Role:        "Project Group–Head of Media & Marketing",
				},
				{
					ID:          19,
					Name:        "Neo Nguyen",
					Email:       ptr("neo.nguyen@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/6afc416374154bfe96cfd818ddbf321a.JPG"),
					LinkedInURL: ptr("https://www.linkedin.com/in/neonguyen-dev/"),
					Role:        "Project Group–Head of Head of Web & UI/UX",
				},
			},
		},
		{
			Name: "Project Manager",
			People: []Person{
				{
					ID:          20,
					Name:        "Smriti Dubey",
					Email:       ptr("a@armada.nu"),
					Picture:     ptr("https://armada-ais-files.s3.eu-north-1.amazonaws.com/profiles/picture_original/e8b0f158bc424933b78b5b453d5c5858.JPG"),
					LinkedInURL: ptr("https://www.linkedin.com/in/smriti-dubey-68227a164/"),
					Role:        "Project Manager–Project Manager",
				},
			},
		},
    }
}
package com.ergulcu.vetpet.controller;

import com.ergulcu.vetpet.dao.OwnerRepository;
import com.ergulcu.vetpet.dao.PetRepository;
import com.ergulcu.vetpet.model.Pet;
import javax.validation.Valid;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.validation.BindingResult;
import org.springframework.validation.ObjectError;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;

/**
 *
 * @author Mehmet ERGÜLCÜ
 */
@Controller
public class PetController {

    @Autowired
    private PetRepository petRepository;
    @Autowired
    private OwnerRepository ownerRepository;

    @GetMapping("/pet/list")
    public String listPet(Pet pet,Model model) {
        model.addAttribute("pets", petRepository.findAll());
        return "pet/list";
    }
    
    @PostMapping("/pet/list")
    public String filterPet(Pet pet, Model model) {
        model.addAttribute("pets", petRepository.findByNameContainingIgnoreCase(pet.getName()));
        return "pet/list";
    }

    @GetMapping("/pet/create")
    public String addPet(Pet pet, Model model) {
        model.addAttribute("owners", ownerRepository.findAll());
        return "pet/create";

    }

    @PostMapping("/pet/create")
    public String addPet(@Valid Pet pet, BindingResult result ,Model model) {
        if (result.hasErrors()) {
            for (ObjectError error : result.getAllErrors()) {
                System.out.println(error);
            }
            System.out.println();
            return "pet/create";
        }
        petRepository.save(pet);
        return "redirect:/pet/list";
    }

    @GetMapping("/pet/update/{id}")
    public String showUpdateForm(@PathVariable("id") long id, Model model) {
        Pet pet = petRepository.findById(id).orElseThrow(() -> new IllegalArgumentException("Invalid pet Id:" + id));
        model.addAttribute("pet", pet);
        model.addAttribute("owners", ownerRepository.findAll());
        return "pet/update";
    }

    @PostMapping("/pet/update/{id}")
    public String updatePet(@PathVariable("id") long id, @Valid Pet pet, BindingResult result) {
        if (result.hasErrors()) {
            pet.setId(id);
            return "pet/update";
        }
        petRepository.save(pet);
        return "redirect:/pet/list";
    }

    @GetMapping("/pet/delete/{id}")
    public String deletePet(@PathVariable("id") long id) {
        Pet pet = petRepository.findById(id).orElseThrow(() -> new IllegalArgumentException("Invalid pet Id:" + id));
        petRepository.delete(pet);
        return "redirect:/pet/list";
    }
}
